package main

import "os"

func main() {
	driver := newDriver()
	if err := driver.Drive(); err != nil {
		os.Exit(1)
	}
}


func lengthOfLongestSubstring(s string) int {
    if len(s) == 0 {
        return 0
    }
    if len(s) == 1 {
        return 1
    }
    b := []byte(s)
    n, bign := 1, 1
    m := make(map[byte]bool)
    m[b[0]] = true
    for i := 1; i < len(b); i++ {
        curr := b[i]
        if _, ok := m[curr]; ok {
            if n > bign {
                bign = n
            }
            n = 1
        } else {
            n += 1
            m[curr] = true
            //prev = curr
        }
    }
    
    if bign < n {
        return n
    } else {
        return bign
    }
}