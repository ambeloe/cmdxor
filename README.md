# cmdxor
xorsearch type program

can try to find a string in a file thats obfuscated with xor

<pre>Usage of cmdxor:
  -C	use as many threads as possible
  -D string
    	folder to dump ALL xored variants
  -K string
    	keyword as a string
  -c uint
    	number of threads to use (default 1)
  -i string
    	input file
  -k string
    	key as a series of csv base 10 bytes
  -m uint
    	max length of key in bytes to try (default 1)
  -n uint
    	minimum number of matches that need to be found
  -o string
    	output file
  -s string
    	search for string in input
  -x	xor input file with key and write to output
</pre>
