package client

import (
	"github.com/philipyao/prpc/registry"
	"fmt"
	"errors"
	"github.com/philipyao/prpc/codec"
)

const (
	noSpecifiedVersion		= ""
	noSpecifiedIndex		= -1
)

type endPoint struct{
	index 		int
	weight		int
	version		string
	styp 		codec.SerializeType
	addr 		string
	conn 		*RPCClient

	//calltimes
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

func (sc *svcClient) Call(serviceMethod string, args interface{}, reply interface{}) error {
	//todo 过滤机制
	ep := sc.selector(sc.endPoints)
	if ep == nil {
		//todo
		return errors.New("no available rpc servers")
	}

	//failover机制
	retry := 3
	var err error
	for retry > 0 {
		//err = ep.rpcClient.Call(serviceMethod, args, reply)
		if err == nil {
			break
		}
		retry--
	}

	return nil
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
	for _, fnOpt := range opts {
		if fnOpt == nil {
			return nil
		}
		fnOpt(sc)
	}
	//设置selector
	cs := configSelect{
		typ: sc.selectType,
		index: sc.index,
	}
	slt, err := createSelector(cs)
	if err != nil {
		//handle err
		return nil
	}
	sc.selector = slt

	//todo 获取endpoints, watch endpoints的变化
	//nodes, err := reg.Subscribe(service, group, sc)
	//提前把version过滤一遍

	return sc
}