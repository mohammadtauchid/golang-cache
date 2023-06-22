package larc

import (
    "fmt"
	"os"
	"time"
    "sort"

	"github.com/mohammadtauchid/golang-cache/v2/simulator"
	"github.com/secnot/orderedmap"
)

type (
    Node struct {
        lba int
        op  string
    }

    LARC struct {
        maxlen      int
        available   int
        hit         int
        miss        int
        wc          int

        q           *orderedmap.OrderedMap
        qr          []int
        cr          int
    }
)

func NewLARC(value int) *LARC {
    return &LARC {
        maxlen:     value,
        available:  value,
        hit:        0,
        miss:       0,
        wc:         0,

        q:      orderedmap.NewOrderedMap(),
        qr:     make([]int, int(0.1 * float64(value))),
        cr:     int(0.1 * float64(value)),
    }
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

func (larc *LARC) Filter(data *Node) (exists bool) {
    if index, ok := getIndex(larc.qr, data.lba); ok {
        if index == len(larc.qr) - 1 {
            larc.qr = larc.qr[:index]
        } else {
            larc.qr = append(larc.qr[:index], larc.qr[index + 1:]...)
        }
        return true
    }

    larc.qr = append(larc.qr, data.lba)

    if len(larc.qr) > larc.cr {
        larc.qr = larc.qr[len(larc.qr) - larc.cr:]
    }
    return false
}

func (larc *LARC) Put(data *Node) (exists bool) {
    // cache hit
    if _, ok := larc.q.Get(data.lba); ok {
        larc.hit++
        larc.q.MoveLast(data.lba)

        // resize qr
        larc.cr = larc.cr - larc.maxlen / (larc.maxlen - larc.cr)
        if float64(larc.cr) < 0.1 * float64(larc.maxlen) {
            larc.cr = int(0.1 * float64(larc.maxlen))
        }
        size := larc.cr
        if size > len(larc.qr) {
            size = len(larc.qr)
        }
        larc.qr = larc.qr[len(larc.qr) - size:]

        return true
    }

    // cache miss
    larc.miss++

    // resize qr
    larc.cr = larc.cr + (larc.maxlen / larc.cr)
    if float64(larc.cr) > 0.9 * float64(larc.maxlen) {
        larc.cr = int(0.9 * float64(larc.maxlen))
    }
    size := larc.cr
    if size > len(larc.qr) {
        size = len(larc.qr)
    }
    larc.qr = larc.qr[len(larc.qr) - size:]

    if !larc.Filter(data) {
        return false
    }

    larc.wc++

    if larc.available > 0 {
        larc.available--
        larc.q.Set(data.lba, data.op)
    } else {
        larc.q.PopFirst()
        larc.q.Set(data.lba, data.op)
    }

    return false
}

func (larc *LARC) Get(trace simulator.Trace) (err error) {
    obj := new(Node)
    obj.lba = trace.Addr
    obj.op = trace.Op
    larc.Put(obj)

    return nil
}

func (larc *LARC) PrintToFile(file *os.File, start time.Time) (err error) {
    file.WriteString(fmt.Sprintf("cache size: %d\n", larc.maxlen))
    file.WriteString(fmt.Sprintf("cache hit: %d\n", larc.hit))
    file.WriteString(fmt.Sprintf("cache miss: %d\n", larc.miss))
    file.WriteString(fmt.Sprintf("cache hit ratio: %.4f%%\n", float64(larc.hit) / float64(larc.hit + larc.miss) * 100))
    file.WriteString(fmt.Sprintf("write count: %d\n", larc.wc))
    file.WriteString(fmt.Sprintf("time execution: %8.4f\n", time.Since(start).Seconds()))

    return nil
}   