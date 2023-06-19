package arc

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

    ARC struct {
        maxlen      int
        // available   int
        hit         int
        miss        int
        p           int // p is the number of pages in T1; adaptation parameter

        t1          *orderedmap.OrderedMap
        t2          *orderedmap.OrderedMap
        b1          *orderedmap.OrderedMap
        b2          *orderedmap.OrderedMap
    }
)

func NewARC(value int) *ARC {
    return &ARC{
        maxlen:         value,
        // available:      value,
        hit:            0,
        miss:           0,
        p:              0,
        t1:             orderedmap.NewOrderedMap(),
        t2:             orderedmap.NewOrderedMap(),
        b1:             orderedmap.NewOrderedMap(),
        b2:             orderedmap.NewOrderedMap(),
    }
}

func (arc *ARC) Replace(data *Node) (err error) {
    t1size := arc.t1.Len()
    _, b2exist := arc.b2.Get(data.lba)
    if t1size > 0 && (t1size > arc.p || (b2exist && t1size == arc.p)) {
        // move LRU of T1 to MRU of B1
        lruKey, lruVal, ok := arc.t1.GetFirst()
        if !ok {
            return err
        }
        arc.t1.Delete(lruKey)
        arc.b1.Set(lruKey, lruVal)
    } else {
        // move LRU of T2 to MRU of B2
        lruKey, lruVal, ok := arc.t2.GetFirst()
        if !ok {
            return err
        }
        arc.t2.Delete(lruKey)
        arc.b2.Set(lruKey, lruVal)
    }
    return nil
}

func (arc *ARC) Put(data *Node) (exists bool) {
    // length of list
    t1size := arc.t1.Len()
    t2size := arc.t2.Len()
    b1size := arc.b1.Len()
    b2size := arc.b2.Len()

    // first case: data is in T1 or T2
    if _, ok := arc.t1.Get(data.lba); ok {
        arc.t1.Delete(data.lba)
        arc.t2.Set(data.lba, data.op)
        arc.hit++
        return true
    } else if _, ok := arc.t2.Get(data.lba); ok {
        arc.t2.MoveLast(data.lba)
        arc.hit++
        return true
    }

    // second case: data is in B1
    if _, ok := arc.b1.Get(data.lba); ok {
        // adaptation
        delta := 1
        if b1size < b2size {
            delta = b2size / b1size
        }

        if arc.p + delta >= arc.maxlen {
            arc.p = arc.maxlen
        } else {
            arc.p += delta
        }

        // call subroutine replace
        err := arc.Replace(data)
        if err != nil {
            return false
        }

        // move data from B1 to T2
        arc.b1.Delete(data.lba)
        arc.t2.Set(data.lba, data.op)

        arc.miss++
        return false
    }

    // third case: data is in B2
    if _, ok := arc.b2.Get(data.lba); ok {
        // adaptation
        delta := 1
        if b2size < b1size {
            delta = b1size / b2size
        }

        if arc.p - delta <= 0 {
            arc.p = 0
        } else {
            arc.p -= delta
        }

        // call subroutine replace
        err := arc.Replace(data)
        if err != nil {
            return false
        }

        // move data from B2 to T2
        arc.b2.Delete(data.lba)
        arc.t2.Set(data.lba, data.op)

        arc.miss++
        return false
    }

    // forth case: data is not in any list
    // * first case: T1 and B1 has exaclty c pages
    if t1size + b1size == arc.maxlen {
        if t1size < arc.maxlen {
            key, _, _ := arc.b1.GetFirst()
            arc.b1.Delete(key)
        } else {
            // B1 is empty
            key, _, _ := arc.t1.GetFirst()
            arc.t1.Delete(key)
        }
    }
    // * second case: T1 and B1 has less than c pages
    if arc.t1.Len() + arc.b1.Len() < arc.maxlen {
        allsize := t1size + t2size + b1size + b2size
        if allsize >= arc.maxlen {
            if allsize == 2 * arc.maxlen {
                key, _, _ := arc.b2.GetFirst()
                arc.b2.Delete(key)
            }
            err := arc.Replace(data)
            if err != nil {
                return false
            }
        }
    }

    arc.miss++
    arc.t1.Set(data.lba, data.op)

    return false
}

func (arc *ARC) Get(trace simulator.Trace) (err error) {
    obj := new(Node)
    obj.lba = trace.Addr
    obj.op = trace.Op
    arc.Put(obj)

    return nil
}

func (arc *ARC) PrintToFile(file *os.File, start time.Time) (err error) {
    file.WriteString(fmt.Sprintf("cache size: %d\n", arc.maxlen))
    file.WriteString(fmt.Sprintf("cache hit: %d\n", arc.hit))
    file.WriteString(fmt.Sprintf("cache miss: %d\n", arc.miss))
    file.WriteString(fmt.Sprintf("cache hit ratio: %.4f%%\n", float64(arc.hit) / float64(arc.hit + arc.miss) * 100))
    file.WriteString(fmt.Sprintf("time execution: %8.4f\n", time.Since(start).Seconds()))

    return nil
}