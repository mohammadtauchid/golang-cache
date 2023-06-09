package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mohammadtauchid/golang-cache/v2/lru"
	"github.com/mohammadtauchid/golang-cache/v2/simulator"
)

func main() {
	var (
        traces      []simulator.Trace = make([]simulator.Trace, 0)
        simulator   simulator.Simulator
        timeStart   time.Time
        out         *os.File
        fs          os.FileInfo
        filePath    string
        outPath     string
        algorithm   string
        err         error
        cacheList   []int
    )

    if len(os.Args) < 4 {
        fmt.Println("Usage: ./main [algorithm] [trace file path] [trace size]...")
        fmt.Println("Example: ./main LRU resource/Financial 1000 2000 3000")
        fmt.Println("Available algorithms:")
        fmt.Println("LRU     : Least Recently Used")
        fmt.Println("LFU     : Least Frequently Used")
        fmt.Println("ARC     : Adaptive Replacement Cache")
        fmt.Println("2Q      : Two Queues")
        fmt.Println("Compare : Compare all algorithms")
        os.Exit(1)
    }

    algorithm = os.Args[1]

    filePath = os.Args[2]
    if fs, err = os.Stat(filePath); os.IsNotExist(err) {
        fmt.Printf("Error: %v does not exist", filePath)
        os.Exit(1)
    }

    cacheList, err = validateTraceSize(os.Args[3:])
    if err != nil {
        fmt.Println(err.Error())
        os.Exit(1)
    }

    traces, err = readFile(filePath)
    if err != nil {
        log.Fatalf("Error reading file: %v", err)
    }

    outPath = fmt.Sprintf("%v_%v_%v.txt", time.Now().Unix(), algorithm, fs.Name())

    out, err = os.Create(fmt.Sprintf("output/%v/%v", algorithm, outPath))
    if err != nil {
        log.Fatalf(err.Error())
    }
    defer out.Close()

    for _, cache := range cacheList {
        switch strings.ToLower(algorithm) {
        case "lru":
            simulator = lru.NewLru(cache)
        }

        timeStart = time.Now()

        for _, trace := range traces {
            err = simulator.Get(trace)
            if err != nil {
                log.Fatal(err.Error())
            }
        }

        simulator.PrintToFile(out, timeStart)
    }

    fmt.Printf("Done")
}

func validateTraceSize(tracesize []string) (sizeList []int, err error) {
    var (
        cacheList   []int
        cache       int
    )

    for _, size := range tracesize {
        if cache, err = strconv.Atoi(size); err != nil {
            fmt.Println("Error: trace size must be an integer")
            return sizeList, err
        }
        cacheList = append(cacheList, cache)
    }

    return cacheList, nil
}

func readFile(filePath string) (traces []simulator.Trace, err error) {
    var (
        file    *os.File
        scanner *bufio.Scanner
        row     []string
        address int
    )

    file, err = os.Open(filePath)
    if err != nil {
        return traces, err
    }
    defer file.Close()

    scanner = bufio.NewScanner(file)

    for scanner.Scan() {
        row = strings.Split(scanner.Text(), ",")
        address, err = strconv.Atoi(row[0])
        
        if err != nil {
            return traces, err
        }

        traces = append(traces, simulator.Trace{
            Addr:   address,
            Op:     row[1],
        })
    }

    return traces, nil
}

// func even(val int) (res bool, err error) {
//     if val % 2 != 0 {
//         return false, nil
//     }
//     return true, nil
// }

// func main() {
// 	// Open the file for writing
// 	file, err := os.OpenFile("output.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
// 	if err != nil {
// 		fmt.Println("Error opening file:", err)
// 		return
// 	}
// 	defer file.Close()

//     res, err := even(3) || even(4)

// 	// String to be written
// 	// str := " : Hello, World!\n"

// 	// Write the string to the file
// 	_, err = file.WriteString(
//         fmt.Sprintf("%s : Failed to move LBA %d to MRU position\n", time.Now().String(), 12),
//     )
// 	if err != nil {
// 		fmt.Println("Error writing to file:", err)
// 		return
// 	}

// 	fmt.Println("String successfully written to file.")
// }
