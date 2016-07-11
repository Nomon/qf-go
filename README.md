# Quotient Filter

a go implementation of the quotient filter data structure. Quotient filter is a
probabilistic data structure similar to a Bloom Filter. As BF it is used to approximate set membership. It never returns false for added keys but has a probability of returning a false positive, the probability depends on the amount of remainder bits and fill rate of the filter.

Its size is 10-20% more than a bloom filter with same FP rate but it usually is faster.
It also has a few additional benefits:
- Filters can be merged without rehashing keys
- Adding or checking a key requires evaluating only a single hash function
- Can be resized without rehashing

For more details, see: https://en.wikipedia.org/wiki/Quotient_filter


## Interface

```go
// Create a filter that can hold 1m elements while maintaining 1% false positive
// rate when at 1 million items length.
qf := NewPropability(1000000, 0.01)
qf.Add("key")
if !qf.Contains("key") {
  panic("False negative not possible")
}
```

## Credits

To all existing quotient filter implementations I used as a base and learning material:
https://github.com/vedantk/quotient-filter
https://github.com/bucaojit/QuotientFilter
https://github.com/dsx724/php-quotient-filter


## License MIT
