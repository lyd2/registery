package registery

import (
	"errors"
	"sync"
	"sync/atomic"
)

const MIN_WEIGHT = 1
const MAX_WEIGHT = 100

// 一个实例
type instance struct {
	// 地址
	host string
	// 权重，用于负载均衡
	weight uint
	// 分段标记
	milestone uint
}

func NewInstance(host string, weight uint) (*instance, error) {
	if weight < MIN_WEIGHT || weight > MAX_WEIGHT {
		return nil, errors.New("Wrong weight")
	}
	return &instance{
		host:      host,
		weight:    weight,
		milestone: 0,
	}, nil
}

func (i *instance) GetHost() string {
	return i.host
}

func (i *instance) SetHost(host string) {
	i.host = host
}

func (i *instance) GetWeight() uint {
	return i.weight
}

func (i *instance) SetWeight(weight uint) error {
	if weight < MIN_WEIGHT || weight > MAX_WEIGHT {
		return errors.New("Wrong weight")
	}
	i.weight = weight
	return nil
}

func (i *instance) getMilestone() uint {
	return i.milestone
}

func (i *instance) setMilestone(start uint) uint {
	i.milestone = start + i.weight
	return i.milestone
}

func (i *instance) before(n uint) bool {
	return i.milestone < n
}

func (i *instance) after(n uint) bool {
	return i.milestone > n
}

func (i *instance) equal(n uint) bool {
	return i.milestone == n
}

// 一个服务
type Service struct {
	// 服务名称
	servName string
	// 实例列表
	instanceList map[string]*instance
	// 映射表，用于计算负载均衡
	segment []*instance
	// 最大的分段值
	mmi uint
	// 总请求次数，用于计算负载均衡
	count uint64
	// rwlock
	lock *sync.RWMutex
}

func NewService(servName string) *Service {
	return &Service{
		servName:     servName,
		instanceList: make(map[string]*instance),
		segment:      []*instance{nil},
		mmi:          0,
		count:        0,
		lock:         &sync.RWMutex{},
	}
}

func (s *Service) GetServName() string {
	return s.servName
}

func (s *Service) HostExists(host string) bool {
	_, ok := s.instanceList[host]
	return ok
}

/**
对实例列表的读写操作是并发进行的，因此需要考虑并发安全
由于对实例的读操作远多于写操作，且读操作是并发安全的，因此使用读写锁实现
*/
func (s *Service) Submit(obj *instance) *Service {

	// 获取写锁
	s.lock.Lock()
	defer s.lock.Unlock()

	// 设置实例列表
	s.instanceList[obj.GetHost()] = obj

	// 构建分段表
	s.buildSegment()

	return s
}

// 删除一个实例
func (s *Service) Delete(host string) bool {

	// 获取写锁
	s.lock.Lock()
	defer s.lock.Unlock()

	// 删除对应实例
	if !s.HostExists(host) {
		return false
	}
	delete(s.instanceList, host)

	// 成功删除则重新构建分段表
	s.buildSegment()

	return true
}

/**
构建分段表
分段表用于计算负载均衡
例如有三个实例，它们对应的权重为： a->1, b->2, c->3
这就表示每6个请求，一个请求到a，两个请求到b，三个请求到c
我们将构建出分段表： a:1, b:3, c:6
*/
func (s *Service) buildSegment() {
	var total uint = 0
	s.segment = s.segment[0:1]

	for _, v := range s.instanceList {
		total = v.setMilestone(total)
		s.segment = append(s.segment, v)
	}

	s.mmi = total
}

/**
获取一个实例，使用负载均衡
例如分段表为： a:1, b:3, c:6
则：第一个请求到a，第二、第三个请求到b，第四、第五、第六个请求到c
之后都如此重复
很显然，我们只需要对总请求数 mod 6 即可
有两个点：
1. 对于6、12、18、... 它们 mod 6 结果为 0，而它们实际上是要走最后一个地址的
2. 在计算总请求数时，由于 i++ 不是线程安全的，使用了 atomic 来原子的自增
*/
func (s *Service) Load() (*instance, error) {

	// 获取读锁
	s.lock.RLock()
	defer s.lock.RUnlock()

	if s.mmi == 0 {
		return nil, errors.New("Service is empty")
	}

	// 获取当前的请求次数，由于最开始为0，因此第一次请求会返回1
	c := atomic.AddUint64(&s.count, 1)

	// 获取待查找的分段
	m := uint(c % uint64(s.mmi))
	if m == 0 {
		m = s.mmi
	}

	// 开始分段查找
	// end 就是找到的 index
	start, end := 0, len(s.segment)-1
	if end == 0 {
		panic("UnknownError")
	}
	for {
		if start == end - 1 {
			break
		}
		index := (start + end) / 2

		if s.segment[index].before(m) {
			start = index
		} else {
			end = index
		}
	}

	return s.segment[end], nil

	/*
		我们使用几个例子来说明最后的查找算法

		假设初始序列为： 0->nil
		则 panic，断言不可能存在这种情况

		假设初始序列为： 0->nil, 1->10
		此时 start = 0, end = 1
		由于满足了 break 的条件，因此跳出循环，end 就是待查找的 index

		假设初始序列为： 0->nil, 1->10, 2->20 ，查找 3 所在的分段
		此时 start = 0, end = 2
		计算得到 index = 1，由于 3 小于 10，因此 end = 1
		由于满足 break 条件，跳出循环，end 就是待查找的 index

		假设初始序列为： 0->nil, 1->10, 2->20 ，查找 13 所在的分段
		此时 start = 0, end = 2
		计算得到 index = 1，由于 13 大于 10，因此 start = 1
		由于满足 break 条件，跳出循环，end 就是待查找的 index

		假设初始序列为： 0->nil, 1->10, 2->20, 3->30, 4->40, 5->50 ，查找 33 所在的分段
		此时 start = 0, end = 5
		计算得到 index = 2，由于 33 大于 20，因此 start = 2
		计算得到 index = 3，由于 33 大于 30，因此 start = 3
		计算得到 index = 4，由于 33 小于 40，因此 end = 4
		由于满足 break 条件，跳出循环，end 就是待查找的 index
	*/

}
