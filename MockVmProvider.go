package main

import (
	"container/list"
	"github.com/google/uuid"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
	samplev1alpha1 "k8s.io/sample-controller/pkg/apis/samplecontroller/v1alpha1"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
)

type VmEntity struct {
	id string
	cpuUtilization string
	name string
	crSpaceName string
}

type VmResponse struct {
	success bool
	code int32
	msg string
	vmEntity *VmEntity
}

type VmListResponse struct {
	success bool
	code int32
	msg string
	vmList *list.List
}

type MockManager struct {
	// vmId -> vmEntity
	inMemoryData map[string]*VmEntity
	// lock
	lock sync.RWMutex
}

func NewMockManager() *MockManager {
	return &MockManager{
		inMemoryData: make(map[string]*VmEntity),
	}
}

/**
201: succeeded in creating VM
409: failed to create VM because of VM name duplication
500: failed to create VM somehow
*/
func (m *MockManager) CreateVm(vm *samplev1alpha1.VM) *VmResponse {
	m.lock.Lock()
	defer m.lock.Unlock()

	//check vm is valid
	inputkey,err := cache.MetaNamespaceKeyFunc(vm)
	if err != nil {
		utilruntime.HandleError(err)
		return &VmResponse{
			success:  false,
			code:     500,
			msg:      "MetaNamespaceKeyFunc failed on CR",
			vmEntity: nil,
		}
	}

	if vm == nil || !m.CheckNameIsOk(vm.Spec.VMName) {
		return &VmResponse{
			success:  false,
			code:     409,
			msg:      "check name failed",
			vmEntity: nil,
		}
	}

	//check if already created
	for _, val := range m.inMemoryData {
		if val.crSpaceName == inputkey {
			return &VmResponse{
				success:  true,
				code:     201,
				msg:      "already created, vmName is " + val.name,
				vmEntity: val,
			}
		}
	}

	//dup name check
	for _, val := range m.inMemoryData {
		if vm.Spec.VMName == val.name {
			return &VmResponse{
				success:  false,
				code:     409,
				msg:      "dup vm name",
				vmEntity: val,
			}
		}
	}

	rand.Seed(time.Now().UnixNano())
	vmEntity := &VmEntity{
		id:             uuid.New().String(),
		cpuUtilization: strconv.Itoa(rand.Intn(100)),
		name: vm.Spec.VMName,
		crSpaceName: inputkey,
	}

	m.inMemoryData[vmEntity.id] = vmEntity

	return &VmResponse{
		success:  true,
		code:     201,
		msg:      "create success",
		vmEntity: vmEntity,
	}
}

/**
200: succeeded in returning vm list
500: failed to return vm somehow
*/
func (m *MockManager) GetVmList() *VmListResponse {
	vmList := list.New()
	for _, v := range m.inMemoryData {
		vmList.PushBack(v)
	}
	return &VmListResponse{
		success: true,
		code:    200,
		msg:     "success",
		vmList:  vmList,
	}
}

/**
200: succeeded in returning vm
404: not found
500: failed to return vm somehow
*/
func (m *MockManager) GetVm(vmId string) *VmResponse {
	val, ok := m.inMemoryData[vmId]
	if !ok {
		return &VmResponse{
			success:  false,
			code:     404,
			msg:      "not found",
			vmEntity: nil,
		}
	}

	rand.Seed(time.Now().UnixNano())
	val.cpuUtilization = strconv.Itoa(rand.Intn(100))

	return &VmResponse{
		success:  true,
		code:     200,
		msg:      "ok",
		vmEntity: val,
	}
}

/**
200: succeeded in getting status
404: not found
500: failed to get status somehow
*/
func (m *MockManager) GetVmStatus(vmId string) *VmResponse {
	val, ok := m.inMemoryData[vmId]
	if !ok {
		return &VmResponse{
			success:  false,
			code:     404,
			msg:      "not found",
			vmEntity: nil,
		}
	}

	return &VmResponse{
		success:  true,
		code:     200,
		msg:      "ok",
		vmEntity: val,
	}
}

/**
204: succeeded in deleting
500: failed to delete
*/
func (m *MockManager) DeleteVm(vmId string) *VmResponse {
	val, _ := m.inMemoryData[vmId]
	//if !ok {
	//	return &VmResponse{
	//		success:  false,
	//		code:     500,
	//		msg:      "failed, not found",
	//		vmEntity: nil,
	//	}
	//}

	delete(m.inMemoryData, vmId)
	return &VmResponse{
		success:  true,
		code:     204,
		msg:      "ok",
		vmEntity: val,
	}
}

/**
200: ok
403: prohibit to use
500: failed somehow
*/
func (m *MockManager) CheckNameIsOk(name string) bool {
	//TODO: concrete logic
	if strings.Contains(name, "political") {
		return false
	}

	return true
}

func (m *MockManager) PrintVmListResponse(vmListResponse *VmListResponse) {
	if vmListResponse == nil {
		klog.Info("nil\n")
		return
	}

	klog.Infof("VmListResponse:%t, %d, %s\n", vmListResponse.success, vmListResponse.code, vmListResponse.msg)
	if vmListResponse.vmList == nil {
		klog.Info("nil\n")
		return
	}

	for e := vmListResponse.vmList.Front(); e != nil; e = e.Next() {
		vmEntity := e.Value.(*VmEntity)
		klog.Infof("uuid:%s cpu:%s name:%s cr:%s",
			vmEntity.id, vmEntity.cpuUtilization, vmEntity.name, vmEntity.crSpaceName)
	}
}
