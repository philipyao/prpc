package client

import "math/rand"

func (c *Client) selectByRandom(group string) *RPCClient {
	//根据weight来随机
	total := 0
	for _, rpc := range c.clientMap {
		if rpc.svc.ID.Group != group {
			continue
		}
		total += rpc.svc.Weight
	}
	if total == 0 {
		return nil
	}
	rd := rand.Intn(total)
	count := 0
	for _, rpc := range c.clientMap {
		if rpc.svc.ID.Group != group {
			continue
		}
		count += rpc.svc.Weight
		if rd >= count {
			continue
		}
		return rpc
	}
	return nil
}

