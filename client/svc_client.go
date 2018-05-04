package client

import (
	"github.com/philipyao/prpc/registry"
	"fmt"
	"errors"
	"github.com/philipyao/prpc/codec"
	"log"
	"encoding/binary"
	"bytes"
	"crypto/sha256"
)

const (
	noSpecifiedVersion		= ""
	noSpecifiedIndex		= -1
)

type endPoint struct{
	key 		string

	index 		int
	weight		int
	version		string
	styp 		codec.SerializeType
	addr 		string
	conn 		*RPCClient

	callTimes 	uint32
	//failtimes
}
//检查注册中心的数据发生变化后，可变数据(可在线更新的)是否发生改变
//不检查不可变数据，如index，addr等
//如果服务提供方的styp addr要修改，肯定会停掉服务，修改然后重新注册的，不会走到这里
func (ep *endPoint) equalTo(node *registry.Node) bool {
	return ep.weight == node.Weight &&
			ep.version == node.Version
}
//更新可变数据(可在线更新的)
func (ep *endPoint) update(node *registry.Node) {
	ep.weight = node.Weight
	ep.version = node.Version
}

type svcClient struct {
	group		string
	service 	string

	//options
	version 	string		//选择特定版本
	index 		int			//选择特定index的endpoint
	selectType  selectType	//选取算法

	selector selector		//选择器
	endPoints []*endPoint

	registry *registry.Registry
}
func (sc *svcClient) Subscribe() error {
	//获取endpoints, watch endpoints的变化
	nodes, err := sc.registry.Subscribe(sc.service, sc.group, sc)
	if err != nil {
		fmt.Printf("suscribe err: %v\n", err)
		return nil
	}
	if len(nodes) > 0 {
		sc.addEndpoint(nodes)
	}
	return nil
}
func (sc *svcClient) Call(serviceMethod string, args interface{}, reply interface{}) error {
	//todo 过滤机制

	var ep *endPoint
	if sc.index >= 0 {
		//指定固定的index
		for _, e := range sc.endPoints {
			if e.index == sc.index {
				ep = e
				break
			}
		}
		if ep == nil {
			return fmt.Errorf("specified index %v not exist", sc.index)
		}
	} else {
		//selector选取算法来选择节点
		var eps []*endPoint

		//1）如果指定了版本，先根据版本过滤可用的节点
		if sc.version != noSpecifiedVersion {
			for _, v := range sc.endPoints {
				if sc.version != v.version {
					continue
				}
				eps = append(eps, v)
			}
		} else {
			eps = sc.endPoints
		}
		//2)selector选取节点
		if len(eps) > 0 {
			ep = sc.selector(eps)
		}
	}
	if ep == nil {
		//todo
		return errors.New("no available rpc servers")
	}

	//failover机制
	retry := 3
	var err error
	for retry > 0 {
		ep.callTimes++
		smethod := fmt.Sprintf("%v.%v", sc.service, serviceMethod)
		err = ep.conn.Call(smethod, args, reply)
		if err == nil {
			break
		}
		retry--
	}
	return err
}

func (sc *svcClient) setVersion(v string) {
	sc.version = v
}

func (sc *svcClient) setIndex(index int) error {
	if index < 0 {
		return errors.New("negtive index not allowed")
	}
	sc.index = index
	return nil
}

func (sc *svcClient) setSelectType(styp selectType) error {
	if int(styp) < 0 || int(styp) > len(selectors) {
		return fmt.Errorf("select type %v not support", styp)
	}
	sc.selectType = styp
	return nil
}

func (sc *svcClient) decorate(opts ...fnOptionService) error {
	for n, fnOpt := range opts {
		if fnOpt == nil {
			return fmt.Errorf("err: decrator service client, nil option no.%v", n + 1)
		}
		err := fnOpt(sc)
		if err != nil {
			return err
		}
	}
	return nil
}

func (sc *svcClient) addEndpoint(nodes []*registry.Node) {
	for _, node := range nodes {
		ep := &endPoint{
			key: node.Path,
			index: node.ID.Index,
			weight: node.Weight,
			version: node.Version,
			styp: codec.SerializeType(node.Styp),
			addr: node.Addr,
		}
		rpc := newRPC(ep)
		if rpc == nil {
			continue
		}
		ep.conn = rpc
		sc.endPoints = append(sc.endPoints, ep)
		fmt.Printf("add endpoint: %+v, total %v\n", ep, len(sc.endPoints))
	}
}

func (sc *svcClient) delEndpoint(dels []string) {
	for _, del := range dels {
		for i, ep := range sc.endPoints {
			if del == ep.key {
				sc.endPoints = append(sc.endPoints[:i], sc.endPoints[i+1:]...)
				fmt.Printf("delete endpoint %+v, total %v\n", ep, len(sc.endPoints))
				break
			}
		}
	}
}

func (sc *svcClient) updateEndpoint(node *registry.Node) {
	for _, ep := range sc.endPoints {
		if ep.key == node.Path {
			//关心的数据确实发生变化
			if !ep.equalTo(node) {
				fmt.Printf("update endpoint<%+v> by node<%+v>\n", ep, node)
				ep.update(node)
			}
			return
		}
	}
	fmt.Printf("node<%+v> update, found no corresponding endpoin\n", node)
}

func (sc *svcClient) hashCode() (string, error) {
	var err error
	endian := binary.LittleEndian
	buf := new(bytes.Buffer)
	for _, v := range []interface{}{int32(sc.index), int32(sc.selectType)} {
		err = binary.Write(buf, endian, v)
		if err != nil {
			fmt.Println("binary.Write failed:", err)
			return "", err
		}
	}
	for _, v := range []string{sc.service, sc.group, sc.version} {
		buf.Write([]byte(v))
	}
	hash := sha256.New()
	hash.Write(buf.Bytes())
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func (sc *svcClient) dumpMetrics() {
	fmt.Println("******* dumpMetrics *******")
	for _, ep := range sc.endPoints {
		fmt.Printf("endpoint: index<%v> weight<%v> callTimes<%v>\n", ep.index, ep.weight, ep.callTimes)
	}
}
//===========================================================

func newSvcClient(service, group string, reg *registry.Registry, opts ...fnOptionService) *svcClient {
	sc := &svcClient{
		group: group,
		service: service,
		registry: reg,
		version: registry.DefaultVersion,		//默认匹配缺省版本
		index: noSpecifiedIndex,				//默认不指定index
		selectType: SelectTypeWeightedRandom,   //默认按照权重随机获得endpoint
	}
	//修饰svcClient
	err := sc.decorate(opts...)
	if err != nil {
		log.Println(err)
		return nil
	}
	//设置selector
	cs := configSelect{
		typ: sc.selectType,
		index: sc.index,
	}
	slt, err := createSelector(cs)
	if err != nil {
		//handle err
		fmt.Printf("createSelector err: %v\n", err)
		return nil
	}
	sc.selector = slt
	return sc
}