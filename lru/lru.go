package lru

import (
	"fmt"
    "os"
    "time"

    "github.com/mohammadtauchid/golang-cache/v2/simulator"
    "github.com/secnot/orderedmap"
)

type (
	Node struct {
		lba int
        op  string
	}

    LRU struct {
        maxlen      int
        available   int
        hit         int
        miss        int

        list        *orderedmap.OrderedMap
    }
)

func NewLru(value int) *LRU {
    return &LRU{
        maxlen:         value,
        available:      value,
        hit:            0,
        miss:           0,
        list:           orderedmap.NewOrderedMap(),
    }
}

func (lru *LRU) Put(data *Node) (exists bool) {
    log, err := os.OpenFile("lru.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    if err != nil {
        fmt.Println("Error opening file:", err)
        return
    }
    defer log.Close()

    if _, _, ok := lru.list.GetLast(); !ok {
        log.WriteString(time.Now().String() + " : LRU cache is empty\n")
    }

    if _, ok := lru.list.Get(data.lba); ok {
        lru.hit++

        if ok := lru.list.MoveLast(data.lba); !ok {
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
        lru.miss++

        if lru.available > 0 {
            lru.available--
        } else {
            evictedLBA, _, _ := lru.list.GetFirst()
            lru.list.Delete(evictedLBA)
        }
        
        lru.list.Set(data.lba, data.op)
        return false
    }   
}

func (lru *LRU) Get(trace simulator.Trace) (err error) {
    obj := new(Node)
    obj.lba = trace.Addr
    obj.op = trace.Op
    lru.Put(obj)

    return nil
}

func (lru *LRU) PrintToFile(file *os.File, start time.Time) (err error) {
    file.WriteString(fmt.Sprintf("cache size: %d\n", lru.maxlen))
    file.WriteString(fmt.Sprintf("cache hit: %d\n", lru.hit))
    file.WriteString(fmt.Sprintf("cache miss: %d\n", lru.miss))
    file.WriteString(fmt.Sprintf("cache hit ratio: %.4f%%\n", float64(lru.hit) / float64(lru.hit + lru.miss) * 100))
    file.WriteString(fmt.Sprintf("time execution: %8.4f\n", time.Since(start).Seconds()))

    return nil
}
