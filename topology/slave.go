package topology

import (
	"github.com/sydnash/lotou/core"
	"github.com/sydnash/lotou/encoding/gob"
	"github.com/sydnash/lotou/log"
	"github.com/sydnash/lotou/network/tcp"
)

type slave struct {
	*core.Base
	*core.EmptyRequest
	*core.EmptyCall
	decoder *gob.Decoder
	encoder *gob.Encoder
	client  uint
}

func StartSlave(ip, port string) {
	m := &slave{Base: core.NewBase()}
	m.decoder = gob.NewDecoder()
	m.encoder = gob.NewEncoder()
	core.RegisterService(m)
	core.Name(m.Id(), ".slave")
	c := tcp.NewClient(ip, port, m.Id())
	m.client = c.Run()
	m.run()
}

func (s *slave) run() {
	s.SetDispatcher(s)
	go func() {
		for msg := range s.In() {
			s.DispatchM(msg)
		}
	}()
}

func (s *slave) NormalMSG(dest, src uint, msgEncode string, data ...interface{}) {
	if msgEncode == "go" {
		//dest is master's id, src is core's id
		//data[0] is cmd such as (registerNode, regeisterName, getIdByName...)
		//data[1] is dest nodeService's id
		//parse node's id, and choose correct agent and send msg to that node.
		t1 := s.encode(data)
		core.Send(s.client, s.Id(), tcp.CLIENT_CMD_SEND, t1)
	} else if msgEncode == "socket" {
		//dest is master's id, src is agent's id
		//data[0] is socket status
		//data[1] is a gob encode data
		//it's first encode value is cmd such as (registerNodeRet, regeisterNameRet, getIdByNameRet, forword...)
		//it's second encode value is dest service's id.
		//find correct agent and send msg to that node.
		cmd := data[0].(int)
		if cmd == tcp.CLIENT_DATA {
			s.decoder.SetBuffer(data[1].([]byte))
			sdata, _ := s.decoder.Decode()
			array := sdata.([]interface{})
			scmd := array[0].(string)
			if scmd == "registerNodeRet" {
				nodeId := array[1].(uint)
				core.RegisterNodeRet(nodeId)
			} else if scmd == "syncName" {
				serviceId := array[1].(uint)
				serviceName := array[2].(string)
				core.SyncName(serviceId, serviceName)
			} else if scmd == "getIdByNameRet" {
				id := array[1].(uint)
				ok := array[2].(bool)
				name := array[3].(string)
				rid := array[4].(uint)
				core.GetIdByNameRet(id, ok, name, rid)
			} else if scmd == "forward" {
				msg := array[1].(*core.Message)
				s.forwardM(msg)
			}
		} else if cmd == tcp.AGENT_CLOSED {
		}
	}
}

func (s *slave) forwardM(msg *core.Message) {
	isLcoal := core.CheckIsLocalServiceId(msg.Dest)
	if isLcoal {
		core.ForwardLocal(msg)
		return
	}
	log.Warn("recv msg not forward to this node.")
}

func (s *slave) encode(d []interface{}) []byte {
	s.encoder.Reset()
	s.encoder.Encode(d)
	s.encoder.UpdateLen()
	t := s.encoder.Buffer()
	//make a copy to be send.
	t1 := make([]byte, len(t))
	copy(t1, t)
	return t1
}

func (s *slave) CloseMSG(dest, src uint) {
	_, _ = dest, src
	s.Close()
}
