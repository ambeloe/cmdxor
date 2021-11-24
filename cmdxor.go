package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strconv"
	"strings"
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
	var dxor = flag.Bool("X", false, "xor input file with key and write to output")
	var search = flag.String("s", "", "search for string in input")
	var bSearch = flag.String("S", "", "search for hex bytes in input (length must be even)")
	var ddxor = flag.String("D", "", "folder to dump ALL xored variants")

	//additional flags for operations
	var keyB = flag.String("k", "", "key as a series of csv base 10 bytes")
	var keyS = flag.String("K", "", "key as a string")
	var nCpu = flag.Uint("c", 0, "number of threads to use, 0 (default) uses as many as possible")
	var minMatch = flag.Uint("n", 0, "minimum number of matches that need to be found")
	var maxLen = flag.Uint("m", 1, "max length of key in bytes to try")
	var hexMode = flag.Bool("x", false, "print positions as hex offsets instead of decimal")

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

	if *nCpu == 0 {
		*nCpu = uint(runtime.NumCPU())
	}

	//xor input file with key
	if *dxor {
		//generate actual key from either string or csv key
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
	//search for xor'ed string or bytes in file
	var IKey []byte
	if *search != "" {
		IKey = []byte(*search)
	} else if *bSearch != "" {
		IKey, err = hex.DecodeString(*bSearch)
		if err != nil {
			fmt.Println("byte search string must be even and only contain 0-9a-fA-F")
			os.Exit(1)
		}
	}
	if IKey != nil {
		fmt.Println("searching for string:", *search)
		fmt.Println("using", *nCpu, "threads")
		fmt.Println("minimum number of matches:", *minMatch)
		fmt.Println("hex output:", *hexMode)

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
					if len(r.pos) >= int(*minMatch) {
						fmt.Println("found keyword at", formatArr(r.pos, *hexMode), "with key", r.key)
					}
				}
				res.ms = make([]match, 0)
			}

			baseKey = ipp(baseKey)

			//trying to use a baseKey longer than the search string is useless with xor
			if len(baseKey) > len(IKey)-1 {
				fmt.Println("key space exhausted for this keyword")
				return
			}
		}
		fmt.Println("max length of key reached")
		return
	}
	//xor input file with a pile of keys and write them all to a folder
	if *ddxor != "" {
		err = os.MkdirAll(*ddxor, 0755)
		if err != nil {
			fmt.Println("error creating directory/ies:", err)
			os.Exit(1)
		}
		baseKey = []byte{0}
		for len(baseKey) < int(*maxLen)+1 {
			fmt.Println("writing", baseKey)
			err = ioutil.WriteFile(*ddxor+string(os.PathSeparator)+a2s(baseKey), xor(iF, baseKey), 0644)
			if err != nil {
				fmt.Println("error writing file:", err)
				os.Exit(1)
			}
			baseKey = ipp(baseKey)
		}
		fmt.Println("done")
		return
	}
}

//create filename from key (not ideal but really don't care)
func a2s(b []byte) string {
	var s = ""
	for _, e := range b {
		s += strconv.Itoa(int(e)) + "_"
	}
	return s[:len(s)-1]
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
			fmt.Println("error parsing supplied key")
			os.Exit(1)
		}
		ret[i] = byte(y)
	}
	return ret
}

//gross task splitter
func jobSplit(t, n int) [][2]int {
	var res2 = make([][2]int, n)
	var r = t % n
	var c = 0
	//too tired don't care
	for i := 0; i < len(res2); i++ {
		res2[i][0] = c
		res2[i][1] = c + (t / n) - 1
		if r > 0 {
			res2[i][1]++
			r--
		}
		c = res2[i][1] + 1
	}
	return res2
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

func formatArr(b []int, t bool) string {
	var s = strings.Builder{}

	var bas = 10
	if t {
		bas = 16
	}

	s.WriteRune('[')
	for _, i := range b {
		s.WriteString(strconv.FormatInt(int64(i), bas))
		s.WriteRune(' ')
	}
	s.WriteRune(']')
	return s.String()
}
