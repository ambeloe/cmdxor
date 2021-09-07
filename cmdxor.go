package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strconv"
	"sync"
)

type results struct {
	m  sync.RWMutex
	ms []match
}

type match struct {
	pos []int
	key []byte
}

var iF []byte
//NOT THE ACTUAL KEY; AN EXTRA ELEMENT IS ADDED BY THE WORKERS
var baseKey []byte
var wg sync.WaitGroup
var res results

func main() {
	var err error

	//mandatory flags
	var inF = flag.String("i", "", "input file")
	var outFile = flag.String("o", "", "output file")

	//operation flags
	var dxor = flag.Bool("x", false, "xor input file with key and write to output")
	var search = flag.String("s", "", "search for string in input")

	//additional flags for operations
	var keyB = flag.String("k", "", "key as a series of csv base 10 bytes")
	var keyS = flag.String("K", "", "key as a string")
	var nCpu = flag.Uint("c", 1, "number of threads to use")
	var maxCpu = flag.Bool("C", false, "use as many threads as possible")
	var minMatch = flag.Uint("n", 0, "minimum number of matches that need to be found")
	var maxLen = flag.Uint("m", 1, "max length of key in bytes to try")

	flag.Parse()

	//check if input file is passed, and open it if it was
	if *inF == "" {
		fmt.Println("no input file specified")
		os.Exit(1)
	} else {
		//deprecated as of 1.16 but this way it'll still build on 1.15 (in os package >1.15)
		iF, err = ioutil.ReadFile(*inF)
		if err != nil {
			fmt.Println("error reading input file:", err)
			os.Exit(1)
		}
	}

	//xor input file with baseKey
	if *dxor {
		//generate actual baseKey from either string or csv baseKey
		if *keyB != "" {
			baseKey = parseKey(*keyB)
		} else if *keyS != "" {
			baseKey = []byte(*keyS)
		} else {
			fmt.Println("key not provided")
			os.Exit(1)
		}

		fmt.Println("using key:", string(baseKey), baseKey)
		err = ioutil.WriteFile(*outFile, xor(iF, baseKey), 0644)
		if err != nil {
			fmt.Println("error writing output file:", err)
			os.Exit(1)
		}
		return
	}
	//search for xored string in file
	if *search != "" {
		if *maxCpu {
			*nCpu = uint(runtime.NumCPU())
		}
		fmt.Println("searching for string:", *search)
		fmt.Println("using", *nCpu, "threads")
		fmt.Println("minimum number of matches:", *minMatch)

		var IKey = []byte(*search)

		//create ranges for each thread
		var split = jobSplit(256, int(*nCpu))

		//remember the appended element to the baseKey
		for len(baseKey) < int(*maxLen) {
			//start all tasks
			for i := 0; i < len(split); i++ {
				wg.Add(1)
				go doJob(split[i], IKey)
			}

			//block until all threads are done
			wg.Wait()

			//print matches, if any
			if len(res.ms) > 0 {
				for _, r := range res.ms {
					if len(r.pos) >= int(*minMatch){
						fmt.Println("found keyword at", r.pos, "with key", r.key)
					}
				}
				res.ms = make([]match, 0)
			}

			baseKey = ipp(baseKey)

			//trying to use a baseKey longer than the search string is useless with xor
			if len(baseKey) > len(IKey) - 1 {
				fmt.Println("key space exhausted for this keyword")
				return
			}
		}
		fmt.Println("max length of key reached")
	}
}

func doJob(j [2]int, k []byte) {
	var lkey = append(baseKey, 0)
	var p []byte
	var s []int
	for i := j[0]; i < j[1]+1; i++ {
		lkey[len(lkey)-1] = byte(i)
		p = xor(k, lkey)
		s = findBytes(iF, p)
		if len(s) > 0 {
			res.m.Lock()
			var t = make([]byte, len(lkey))
			copy(t, lkey)
			res.ms = append(res.ms, match{pos: s, key: t})
			res.m.Unlock()
		}
	}
	wg.Done()
}

//xor in slice with baseKey slice
func xor(in, key []byte) []byte {
	var out = make([]byte, len(in))
	var i, iL, kL = 0, len(in), len(key)
	for i < iL {
		out[i] = in[i] ^ key[i%kL]
		i++
	}
	return out
}

//yoinked from parsecmd
func parseKey(s string) []byte {
	var sret = make([]string, 0)
	var x = -1
	var rs = append([]rune(s), ',')
	for i, r := range rs {
		if r != ',' && x == -1 {
			x = i
		} else if r == ',' && x != -1 {
			sret = append(sret, s[x:i])
			x = -1
		}
	}
	var ret = make([]byte, len(sret))
	for i := 0; i < len(sret); i++ {
		y, err := strconv.ParseUint(sret[i], 10, 8)
		if err != nil {
			fmt.Println("error parsing supplied baseKey")
			os.Exit(1)
		}
		ret[i] = byte(y)
	}
	return ret
}

//gross task splitter
func jobSplit(t, n int) [][2]int {
	var res = make([][2]int, n)
	var r = t % n
	var c = 0
	//too tired dont care
	for i := 0; i < len(res); i++ {
		res[i][0] = c
		res[i][1] = c + (t / n) - 1
		if r > 0 {
			res[i][1]++
			r--
		}
		c = res[i][1] + 1
	}
	return res
}

//add 1 to array
func ipp(a []byte) []byte {
	if len(a) == 0 {
		return []byte{1}
	}
	for i := 0; i < len(a); i++ {
		a[i]++
		if a[i] != 0 {
			break
		}
		if i+1 == len(a) {
			a = append(a, 1)
			break
		}
	}
	return a
}

//find positions of pattern in s
func findBytes(s []byte, pattern []byte) []int {
	var r = make([]int, 0)
	for x := 0; x < len(s)-(len(pattern)-1); x++ {
		for y := 0; y < len(pattern); y++ {
			if s[x+y] != pattern[y] {
				goto fug
			}
		}
		r = append(r, x)
	fug:
	}
	return r
}