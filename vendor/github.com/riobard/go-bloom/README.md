Bloom Filter using double hashing.


```go
func doubleFNV(b []byte) (uint64, uint64) {
    hx := fnv.New64()
    hx.Write(b)
    x := hx.Sum64()
    hy := fnv.New64a()
    hy.Write(b)
    y := hy.Sum64()
    return x, y
}

bf := bloom.New(1000000, 0.0001, doubleFNV)
bf.Add([]byte("hello"))
bf.Test([]byte("hello"))
bf.Test([]byte("world"))
```