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

	fmt.Println(service)
}

func TestService_Delete(t *testing.T) {

}

func TestService_Load(t *testing.T) {

}