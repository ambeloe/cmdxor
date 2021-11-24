# cmdxor
xorsearch type program

can try to find a string in a file thats obfuscated with xor

<pre>Usage of ./cmdxor:
  -D string
        folder to dump ALL xored variants
  -K string
        key as a string
  -S string
        search for hex bytes in input (length must be even)
  -X    xor input file with key and write to output
  -c uint
        number of threads to use, 0 (default) uses as many as possible
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
  -x    print positions as hex offsets instead of decimal

</pre>
