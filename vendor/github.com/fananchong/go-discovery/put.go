package godiscovery

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/coreos/etcd/clientv3"
)

type IPut interface {
	INode
	GetPutData() (string, error)
	NewNodeId() (uint32, error)
}

type Put struct {
	Derived   IPut
	nodeId    string
	Id        uint32
	ctx       context.Context
	ctxCancel context.CancelFunc
	nodeIP    string
}

func (this *Put) Open(root context.Context, nodeType int, putInterval int64) error {
	this.ctx, this.ctxCancel = context.WithCancel(root)
	u, err := this.Derived.NewNodeId()
	if err != nil {
		return err
	}
	this.nodeId = fmt.Sprintf("%d-%d", nodeType, u)
	this.Derived.SetId(u)
	xlog.Infoln("node id:", this.nodeId)
	go this.put(nodeType, putInterval)
	return nil
}

func (this *Put) put(nodeType int, putInterval int64) {
	defer func() {
		if err := recover(); err != nil {
			xlog.Errorln("[except] ", err, "\n", string(debug.Stack()))
			if this.Derived.GetClient() != nil {
				go this.put(nodeType, putInterval)
			}
		}
	}()
	tick := time.NewTicker(time.Duration(putInterval) * time.Second)
	for {
		select {
		case <-tick.C:
			cli := this.Derived.GetClient()
			if cli == nil {
				return
			}
			resp, err := cli.Grant(this.ctx, putInterval+5)
			if err != nil {
				xlog.Errorln(err)
			} else {
				var data string
				data, err = this.Derived.GetPutData()
				if err == nil {
					_, err = cli.Put(this.ctx, this.nodeId, this.nodeIP+"#"+data, clientv3.WithLease(resp.ID))
					if err != nil {
						xlog.Errorln(err)
					}
				}
			}
		case <-this.ctx.Done():
			return
		}
	}
}
