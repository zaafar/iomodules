// Copyright 2016 PLUMgrid
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hover

import (
	"fmt"
	"syscall"

	"github.com/vishvananda/netlink"
)

type InterfaceNode interface {
	Node
	Link() netlink.Link
}

type ExtInterface struct {
	NodeBase
	link netlink.Link
}

func NewExtInterface(link netlink.Link) *ExtInterface {
	return &ExtInterface{
		NodeBase: NewNodeBase(-1, -1, link.Attrs().Name, "i/", 1),
		link:     link,
	}
}

func (ifc *ExtInterface) FD() int {
	if ifc.fd >= 0 {
		return ifc.fd
	}
	cflags := []string{
		fmt.Sprintf("-DINTERFACE_ID=%d", ifc.link.Attrs().Index),
	}
	bpf := NewBpfModule(netdevTxC, cflags)
	if bpf == nil {
		panic(fmt.Errorf("Failed to compile bpf module for %s egress", ifc.Path()))
	}
	// free the llvm memory, just keep the fd
	defer bpf.Close()
	fd, err := bpf.LoadNet("egress")
	if err != nil {
		panic(err)
	}
	fd2, err := syscall.Dup(fd)
	if err != nil {
		panic(err)
	}
	ifc.fd = fd2
	return ifc.fd
}

func (ifc *ExtInterface) Link() netlink.Link { return ifc.link }
func (ifc *ExtInterface) SetID(id int)       { ifc.id = id }

type IngressChain struct {
	fd int
}

func NewIngressChain(chain [3]int) (*IngressChain, error) {
	cflags := []string{
		fmt.Sprintf("-DCHAIN_VALUE0=0x%x", chain[0]),
		fmt.Sprintf("-DCHAIN_VALUE1=0x%x", chain[1]),
		fmt.Sprintf("-DCHAIN_VALUE2=0x%x", chain[2]),
	}
	//Debug.Printf("netdev: %v\n", cflags)
	bpf := NewBpfModule(netdevRxC, cflags)
	if bpf == nil {
		return nil, fmt.Errorf("could not compile bpf module for external interface")
	}
	defer bpf.Close()
	fd, err := bpf.LoadNet("ingress")
	if err != nil {
		return nil, err
	}
	fd2, err := syscall.Dup(fd)
	if err != nil {
		return nil, err
	}
	return &IngressChain{fd: fd2}, nil
}

func (c *IngressChain) Close()  { syscall.Close(c.fd) }
func (c *IngressChain) FD() int { return c.fd }