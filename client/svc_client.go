package client

import (
	"github.com/philipyao/prpc/registry"
	"fmt"
	"errors"
)

type endPoint struct{
	index 		int
	weight		int
	version		string
	addr 		string
	conn 		*RPCClient

	//calltimes
	//failtimes
}

type svcClient struct {
	group		string
	service 	string

	//options
	version 	string		//特定版本
	index 		int			//特定index的endpoint
	selectType  selectType	//

	selector selector
	endPoints []*endPoint

	registry *registry.Registry
}

func (sc *svcClient) Call(serviceMethod string, args interface{}, reply interface{}) error {
	//todo 过滤机制
	ep := sc.selector(sc.endPoints)
	if ep == nil {
		//todo
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

// svcClient要监听registry的对于service的事件
func (sc *svcClient) watch() {
	//如果目录group.service没有
	//则可以新建（永久节点。todo 如果group没有了，怎么删除？）

	//version过滤
}

//===========================================================

func newSvcClient(group, service string, registry *registry.Registry, opts ...fnOptionService) *svcClient {
	sc := &svcClient{
		group: group,
		service: service,
		registry: registry,
		version: "",	//默认不指定版本
		index: -1,		//默认不指定index
		selectType: SelectTypeWeightedRandom,   //默认按照权重随机获得endpoint
	}
	//修饰svcClient
	for _, fnOpt := range opts {
		fnOpt(sc)
	}
	//设置selector
	cs := configSelect{
		typ: sc.selectType,
		index: sc.index,
	}
	sc.selector = createSelector(cs)

	//registry.GetService获取endpoints
	//提前把version过滤一遍

	//todo watch endpoints的变化

	return sc
}