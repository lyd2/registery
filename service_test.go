package registery

import (
	"fmt"
	"testing"
)

func TestService_Submit(t *testing.T) {

	service := NewService("test")

	port8000, _ := NewInstance("127.0.0.1:8000", 2)
	port8001, _ := NewInstance("127.0.0.1:8001", 4)
	port8002, _ := NewInstance("127.0.0.1:8002", 6)
	port8003, _ := NewInstance("127.0.0.1:8003", 8)

	service.Submit(port8000).Submit(port8001).Submit(port8002).Submit(port8003)

	if service.mmi != 20 {
		t.Errorf("error")
	}
}

func TestService_Delete(t *testing.T) {
	service := NewService("test")

	port8000, _ := NewInstance("127.0.0.1:8000", 2)
	port8001, _ := NewInstance("127.0.0.1:8001", 4)
	port8002, _ := NewInstance("127.0.0.1:8002", 6)
	port8003, _ := NewInstance("127.0.0.1:8003", 8)
	port8004, _ := NewInstance("127.0.0.1:8004", 10)

	service.Submit(port8000).
		Submit(port8001).
		Submit(port8002).
		Submit(port8003).
		Submit(port8004)

	service.Delete(port8000.GetHost())
	service.Delete(port8003.GetHost())

	if service.mmi != 20 {
		t.Errorf("error 1")
	}

	if len(service.segment) != 4 {
		t.Errorf("error 2")
	}
}

func TestService_Load(t *testing.T) {
	service := NewService("test")

	port8000, _ := NewInstance("127.0.0.1:8000", 2)
	port8001, _ := NewInstance("127.0.0.1:8001", 4)
	port8002, _ := NewInstance("127.0.0.1:8002", 6)
	port8003, _ := NewInstance("127.0.0.1:8003", 8)
	port8004, _ := NewInstance("127.0.0.1:8004", 10)

	service.Submit(port8000).
		Submit(port8001).
		Submit(port8002).
		Submit(port8003).
		Submit(port8004)

	service.Delete(port8000.GetHost())
	service.Delete(port8003.GetHost())

	res := map[string]int{}
	for i := 0; i < 20; i++ {
		instance, _ := service.Load()
		fmt.Println(service.count, instance.GetHost(), instance.GetWeight(), instance.getMilestone())
		res[instance.GetHost()]++
	}

	if res["127.0.0.1:8000"] != 0 {
		t.Errorf("error %s", "127.0.0.1:8000")
	}
	if res["127.0.0.1:8001"] != 4 {
		t.Errorf("error %s", "127.0.0.1:8001")
	}
	if res["127.0.0.1:8002"] != 6 {
		t.Errorf("error %s", "127.0.0.1:8002")
	}
	if res["127.0.0.1:8003"] != 0 {
		t.Errorf("error %s", "127.0.0.1:8003")
	}
	if res["127.0.0.1:8004"] != 10 {
		t.Errorf("error %s", "127.0.0.1:8004")
	}
}
