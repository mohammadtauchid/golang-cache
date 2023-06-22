package marc

import (
	"fmt"
	"os"
	"time"
    "sort"
    // "math"

	"github.com/mohammadtauchid/golang-cache/v2/simulator"
	"github.com/secnot/orderedmap"
)

type (
    Node struct {
        lba int
        op  string
    }

    mARC struct {
        maxlen      int
        // available   int
        hit         int
        miss        int
        p           int // p is the number of pages in T1; adaptation parameter
        wc          int

        state        string // state is the current state of the cache
        hitState     int // hrState is the hit rate of the current state
        hitSample    int // hrSample is the hit rate of the current sample
        hitSampleFil int // hrSampleFil is the filtered hit rate of the current sample
        counter      int
        filCounter   int

        t1          *orderedmap.OrderedMap
        t2          *orderedmap.OrderedMap
        b1          *orderedmap.OrderedMap
        b2          *orderedmap.OrderedMap
        filter      []int
        filSize     int
    }
)

func NewMARC(value int) *mARC {
    return &mARC{
        maxlen:         value,
        // available:      value,
        hit:            0,
        miss:           0,
        p:              0,
        wc:             0,
        state:          "unstable",
        hitState:       0,
        hitSample:      0,
        hitSampleFil:   0,
        counter:        0,
        filCounter:     0,
        t1:             orderedmap.NewOrderedMap(),
        t2:             orderedmap.NewOrderedMap(),
        b1:             orderedmap.NewOrderedMap(),
        b2:             orderedmap.NewOrderedMap(),
        filter:         make([]int, int(0.1 * float64(value))),
        filSize:        int(0.1 * float64(value)),
    }
}

// state changer for mARC
func (marc *mARC) StateChange() (reset bool) {
    hrState := float64(marc.hitState) / float64(marc.counter)
    hrSample := float64(marc.hitSample) / float64(marc.maxlen)
    hrSampleFil := float64(marc.hitSampleFil) / float64(marc.filCounter)

    // fmt.Println(marc.state, marc.hitState, marc.hitSample, marc.hitSampleFil, marc.counter, marc.filCounter)

    if marc.state == "stable" {
        if hrSample < 0.9 * hrState { // changed
            marc.state = "unstable"
            return true
        }
    } else if marc.state == "unstable" {
        if (hrSample >= 0.9 * hrState && hrSample <= 1.1 * hrState) ||
            (hrSample >= 1.2 * hrState && hrSample > 0.2) {
            marc.state = "stable"
        } else if 0.5 * hrState > hrSample || hrSample < 0.1 {
            marc.state = "unique-access"
        }
        return false
    } else {
        if 0.1 * hrSample > hrSampleFil || hrSample > 0.1 {
            marc.state = "unstable"
            return true
        }
    }
    return false
}

func getIndex(slice []int, target int) (index int, ok bool) {
    index = sort.Search(len(slice), func(i int) bool {
        return slice[i] >= target
    })

    if index < len(slice) && slice[index] == target {
        return index, true
    }

    return -1, false
}

func (marc *mARC) Filter(data *Node) (exists bool) {
    marc.filCounter++
    if index, ok := getIndex(marc.filter, data.lba); ok {
        marc.hitSampleFil++
        if index == len(marc.filter) - 1 {
            marc.filter = marc.filter[:index] 
        } else {
            marc.filter = append(marc.filter[:index], marc.filter[index + 1:]...)
        }
        return true
    }

    marc.filter = append(marc.filter, data.lba)

    if len(marc.filter) > marc.filSize {
        marc.filter = marc.filter[len(marc.filter) - marc.filSize:]
    }
    return false
}


func (marc *mARC) Replace(data *Node) (err error) {
    t1size := marc.t1.Len()
    _, b2exist := marc.b2.Get(data.lba)
    if t1size > 0 && (t1size > marc.p || (b2exist && t1size == marc.p)) {
        // move LRU of T1 to MRU of B1
        lruKey, lruVal, ok := marc.t1.GetFirst()
        if !ok {
            return err
        }
        marc.t1.Delete(lruKey)
        marc.b1.Set(lruKey, lruVal)
    } else {
        // move LRU of T2 to MRU of B2
        lruKey, lruVal, ok := marc.t2.GetFirst()
        if !ok {
            return err
        }
        marc.t2.Delete(lruKey)
        marc.b2.Set(lruKey, lruVal)
    }
    return nil
}

func (marc *mARC) Put(data *Node) (exists bool) {
    // length of list
    t1size := marc.t1.Len()
    t2size := marc.t2.Len()
    b1size := marc.b1.Len()
    b2size := marc.b2.Len()

    marc.counter++

    // first case: data is in T1 or T2
    if _, ok := marc.t1.Get(data.lba); ok {
        marc.t1.Delete(data.lba)
        marc.t2.Set(data.lba, data.op)
        marc.hit++
        marc.hitState++
        marc.hitSample++
        
        // resize the filter
        marc.filSize = marc.filSize - marc.maxlen / (marc.maxlen - marc.filSize)
        if float64(marc.filSize) < 0.1 * float64(marc.maxlen) {
            marc.filSize = int(0.1 * float64(marc.maxlen))
        }
        size := marc.filSize
        if size > len(marc.filter) {
            size = len(marc.filter)
        }
        marc.filter = marc.filter[len(marc.filter) - size:]

        return true
    } else if _, ok := marc.t2.Get(data.lba); ok {
        marc.t2.MoveLast(data.lba)
        marc.hit++
        marc.hitState++
        marc.hitSample++

        // resize the filter
        marc.filSize = marc.filSize - marc.maxlen / (marc.maxlen - marc.filSize)
        if float64(marc.filSize) < 0.1 * float64(marc.maxlen) {
            marc.filSize = int(0.1 * float64(marc.maxlen))
        }
        size := marc.filSize
        if size > len(marc.filter) {
            size = len(marc.filter)
        }
        marc.filter = marc.filter[len(marc.filter) - size:]
        
        return true
    }

    // filter data when "stable" or "unique-access"
    if marc.state != "unstable" {
        // resize the filter
        marc.filSize = marc.filSize + (marc.maxlen / marc.filSize)
        if float64(marc.filSize) > 0.9 * float64(marc.maxlen) {
            marc.filSize = int(0.9 * float64(marc.maxlen))
        }
        size := marc.filSize
        if size > len(marc.filter) {
            size = len(marc.filter)
        }
        marc.filter = marc.filter[len(marc.filter) - size:]

        if !marc.Filter(data) {
            marc.miss++
            return false
        }
    }

    marc.wc++

    // second case: data is in B1
    if _, ok := marc.b1.Get(data.lba); ok {
        // adaptation
        delta := 1
        if b1size < b2size {
            delta = b2size / b1size
        }

        if marc.p + delta >= marc.maxlen {
            marc.p = marc.maxlen
        } else {
            marc.p += delta
        }

        // call subroutine replace
        err := marc.Replace(data)
        if err != nil {
            return false
        }

        // move data from B1 to T2
        marc.b1.Delete(data.lba)
        marc.t2.Set(data.lba, data.op)

        marc.miss++
        return false
    }

    // third case: data is in B2
    if _, ok := marc.b2.Get(data.lba); ok {
        // adaptation
        delta := 1
        if b2size < b1size {
            delta = b1size / b2size
        }

        if marc.p - delta <= 0 {
            marc.p = 0
        } else {
            marc.p -= delta
        }

        // call subroutine replace
        err := marc.Replace(data)
        if err != nil {
            return false
        }

        // move data from B2 to T2
        marc.b2.Delete(data.lba)
        marc.t2.Set(data.lba, data.op)

        marc.miss++
        return false
    }

    // forth case: data is not in any list
    // * first case: T1 and B1 has exaclty c pages
    if t1size + b1size == marc.maxlen {
        if t1size < marc.maxlen {
            key, _, _ := marc.b1.GetFirst()
            marc.b1.Delete(key)
        } else {
            // B1 is empty
            key, _, _ := marc.t1.GetFirst()
            marc.t1.Delete(key)
        }
    }
    // * second case: T1 and B1 has less than c pages
    if marc.t1.Len() + marc.b1.Len() < marc.maxlen {
        allsize := t1size + t2size + b1size + b2size
        if allsize >= marc.maxlen {
            if allsize == 2 * marc.maxlen {
                key, _, _ := marc.b2.GetFirst()
                marc.b2.Delete(key)
            }
            err := marc.Replace(data)
            if err != nil {
                return false
            }
        }
    }

    marc.miss++
    marc.t1.Set(data.lba, data.op)

    return false
}

func (marc *mARC) Get(trace simulator.Trace) (err error) {
    obj := new(Node)
    obj.lba = trace.Addr
    obj.op = trace.Op
    marc.Put(obj)

    // state changer
    if marc.counter % marc.maxlen == 0 && marc.counter != 0 {
        // fmt.Println(marc.state)
        if marc.counter >= marc.maxlen * 2 {
            if marc.StateChange() {
                marc.counter = 0
                marc.hitState = 0
            }
        }

        marc.hitSample = 0
        marc.hitSampleFil = 0
        marc.filCounter = 0
    }
    return nil
}

func (marc *mARC) PrintToFile(file *os.File, start time.Time) (err error) {
    file.WriteString(fmt.Sprintf("cache size: %d\n", marc.maxlen))
    file.WriteString(fmt.Sprintf("cache hit: %d\n", marc.hit))
    file.WriteString(fmt.Sprintf("cache miss: %d\n", marc.miss))
    file.WriteString(fmt.Sprintf("cache hit ratio: %.4f%%\n", float64(marc.hit) / float64(marc.hit + marc.miss) * 100))
    file.WriteString(fmt.Sprintf("cache write count: %d\n", marc.wc))
    file.WriteString(fmt.Sprintf("time execution: %8.4f\n", time.Since(start).Seconds()))

    return nil
}