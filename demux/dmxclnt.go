package demux

import (
	"bufio"
	"fmt"

	db "sigmaos/debug"
	"sigmaos/frame"
	"sigmaos/sessp"
	// sp "sigmaos/sigmap"
	"sync"
)

type rpc struct {
	ch      chan error
	seqno   sessp.Tseqno
	request []byte
	reply   []byte
}

func (r *rpc) String() string {
	return fmt.Sprintf("{rpc %d %d %d}", r.seqno, len(r.request), len(r.reply))
}

type rpcMap struct {
	sync.Mutex
	rpcs map[sessp.Tseqno]*rpc
}

func newRpcMap() *rpcMap {
	return &rpcMap{rpcs: make(map[sessp.Tseqno]*rpc)}
}

func (rm *rpcMap) Put(seqno sessp.Tseqno, rpc *rpc) {
	rm.Lock()
	defer rm.Unlock()
	rm.rpcs[seqno] = rpc
}

func (rm *rpcMap) Remove(seqno sessp.Tseqno) (*rpc, bool) {
	rm.Lock()
	defer rm.Unlock()

	if rpc, ok := rm.rpcs[seqno]; ok {
		delete(rm.rpcs, seqno)
		return rpc, true
	}
	return nil, false
}

type DemuxClnt struct {
	out    *bufio.Writer
	in     *bufio.Reader
	seqno  sessp.Tseqno
	rpcmap *rpcMap
	rpcs   chan *rpc
}

func NewDemuxClnt(out *bufio.Writer, in *bufio.Reader) *DemuxClnt {
	dmx := &DemuxClnt{out, in, 0, newRpcMap(), make(chan *rpc)}
	go dmx.reader()
	go dmx.writer()
	return dmx
}

func (dmx *DemuxClnt) writer() {
	for {
		rpc, ok := <-dmx.rpcs
		if !ok {
			db.DPrintf(db.DEMUXCLNT, "%v writer: replies closed\n")
			return
		}
		if err := frame.WriteSeqno(rpc.seqno, dmx.out); err != nil {
			db.DFatalf("writer: WriteSeqno err %v\n", err)
		}
		if err := frame.WriteFrame(dmx.out, rpc.request); err != nil {
			db.DFatalf("writer: WriteFrame err %v\n", err)
		}
		error := dmx.out.Flush()
		if error != nil {
			db.DFatalf("Flush error %v\n", error)
		}
	}
}

func (dmx *DemuxClnt) reader() {
	for {
		seqno, err := frame.ReadSeqno(dmx.in)
		if err != nil {
			db.DPrintf(db.DEMUXCLNT, "reader: ReadSeqno err %v\n", err)
			break
		}
		reply, err := frame.ReadFrame(dmx.in)
		if err != nil {
			db.DPrintf(db.DEMUXCLNT, "reader: ReadFrame err %v\n", err)
			break
		}
		db.DPrintf(db.DEMUXCLNT, "reader: reply %v\n", seqno)
		rpc, ok := dmx.rpcmap.Remove(seqno)
		if !ok {
			db.DFatalf("Remove err %v\n", seqno)
		}
		rpc.reply = reply
		rpc.ch <- nil
	}
}

func (dmx *DemuxClnt) SendReceive(a []byte) ([]byte, error) {
	seqp := &dmx.seqno
	s := seqp.Next()
	rpc := &rpc{request: a, seqno: s, ch: make(chan error)}
	db.DPrintf(db.DEMUXCLNT, "SendReceive: enqueue %v\n", rpc)
	dmx.rpcmap.Put(s, rpc)
	dmx.rpcs <- rpc
	err := <-rpc.ch
	db.DPrintf(db.DEMUXCLNT, "SendReceive: return %v\n", rpc)
	return rpc.reply, err
}
