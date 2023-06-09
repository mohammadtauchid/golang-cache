package lfu

import (
	"fmt"
	"os"
	"time"

	"github.com/mohammadtauchid/golang-cache/v2/simulator"
	"github.com/secnot/orderedmap"
)

type (
	Node struct {
        lba     int
        op      string
        freq    int
    }

    LFU struct {
        maxlen      int
        available   int
        hit         int
        miss        int

        list        *orderedmap.OrderedMap  
    }
)

func NewLFU(value int) *LFU {
    return &LFU{
        maxlen:         value,
        available:      value,
        hit:            0,
        miss:           0,
        list:           orderedmap.NewOrderedMap(),
    }
}

func (lfu *LFU) Put(data *Node) (exists bool) {
    log, err := os.OpenFile("lfu.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    if err != nil {
        fmt.Println("Error opening file:", err)
        return
    }
    defer log.Close()

    if _, _, ok := lfu.list.GetLast(); !ok {
        log.WriteString(time.Now().String() + " : LFU cache is empty\n")
    }

    if _, ok := lfu.list.Get(data.lba); ok {
        lfu.hit++
        data.freq++

        if ok := lfu.list.MoveLast(data.lba); !ok {
            log.WriteString(
                fmt.Sprintf(
                    "%s : Failed to move LBA %d to MRU position\n", 
                    time.Now().String(), 
                    data.lba,
                ),
            )
        }

        return true
    } else {
        lfu.miss++
        data.freq = 1

        if lfu.available > 0 {
            lfu.available--
        } else {
            minFreq := 0
            evictedLBA := 0

            iter := lfu.list.Iter()
            for key, _, ok := iter.Next(); ok; key, _, ok = iter.Next() {
				item, _ := lfu.list.Get(key)
				node := item.(*Node)
				if minFreq == -1 || node.freq < minFreq {
					minFreq = node.freq
					evictedLBA = key.(int)
				}
			}
            lfu.list.Delete(evictedLBA)
        }

        lfu.list.Set(data.lba, data)
        return false
    }
}

func (lfu *LFU) Get(trace simulator.Trace) (err error) {
    obj := new(Node)
    obj.lba = trace.Addr
    obj.op = trace.Op
    lfu.Put(obj)

    return nil
}

func (lfu *LFU) PrintToFile(file *os.File, start time.Time) (err error) {
    file.WriteString(fmt.Sprintf("cache size: %d\n", lfu.maxlen))
    file.WriteString(fmt.Sprintf("cache hit: %d\n", lfu.hit))
    file.WriteString(fmt.Sprintf("cache miss: %d\n", lfu.miss))
    file.WriteString(fmt.Sprintf("cache hit ratio: %.4f%%\n", float64(lfu.hit) / float64(lfu.hit + lfu.miss) * 100))
    file.WriteString(fmt.Sprintf("time execution: %8.4f\n", time.Since(start).Seconds()))

    return nil
}
