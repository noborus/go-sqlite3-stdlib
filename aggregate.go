package stdlib

import (
	"bytes"
	"math"
	"sort"

	"gonum.org/v1/gonum/stat"
)

type mode struct {
	counts   map[any]int
	top      any
	topCount int
}

func newMode() *mode {
	return &mode{
		counts: map[any]int{},
	}
}

func (m *mode) Step(x any) {
	m.counts[x]++
	c := m.counts[x]
	if c > m.topCount {
		m.top = x
		m.topCount = c
	}
}

func (m *mode) Done() any {
	return m.top
}

type stddev struct {
	xs []float64
}

func newStddev() *stddev { return &stddev{} }

func (s *stddev) Step(x any) {
	s.xs = append(s.xs, floaty(x))
}

func (s *stddev) Done() float64 {
	return stat.PopStdDev(s.xs, nil)
}

type percentile struct {
	xs         []float64
	percentile float64
}

func newPercentile() *percentile { return &percentile{} }

func newPercentileN(n int) func() *percentile {
	return func() *percentile {
		p := newPercentile()
		p.percentile = float64(n)
		return p
	}
}

func (s *percentile) Step(x any, perc ...any) {
	if len(perc) > 0 {
		s.percentile = floaty(perc[0])
	}

	s.xs = append(s.xs, floaty(x))
}

func (s *percentile) Done() float64 {
	if s.percentile == 0 || len(s.xs) == 0 {
		return 0
	}

	sort.Float64s(s.xs)
	r := stat.Quantile(s.percentile/100, stat.Empirical, s.xs, nil)
	return r
}

type sqliteValueKind uint

const (
	sqliteNull sqliteValueKind = iota
	sqliteInt
	sqliteString
	sqliteReal
	sqliteBlob
)

type sqliteValue struct {
	kind sqliteValueKind
	i    int64
	s    string
	r    float64
	b    []byte
}

type sqliteValues []sqliteValue

func (svs *sqliteValues) Len() int {
	return len(*svs)
}

func (svs *sqliteValues) Less(i, j int) bool {
	ie := (*svs)[i]
	je := (*svs)[j]
	if ie.kind != je.kind {
		// TODO: support mixed value types?
		return false
	}

	switch ie.kind {
	case sqliteInt:
		return ie.i < je.i
	case sqliteString:
		return ie.s < je.s
	case sqliteReal:
		return ie.r < je.r
	case sqliteBlob:
		return bytes.Compare(ie.b, je.b) < 0
	}

	return false
}

func (svs *sqliteValues) Swap(i, j int) {
	(*svs)[i], (*svs)[j] = (*svs)[j], (*svs)[i]
}

type median struct {
	xs sqliteValues
}

func newMedian() *median {
	return &median{}
}

func (m *median) Step(x any) {
	v := sqliteValue{kind: sqliteNull}
	switch t := x.(type) {
	case int64:
		v.kind = sqliteInt
		v.i = t
	case int:
		v.kind = sqliteInt
		v.i = int64(t)
	case string:
		v.kind = sqliteString
		v.s = t
	case float64:
		v.kind = sqliteReal
		v.r = t
	case []byte:
		v.kind = sqliteBlob
		v.b = t
	}
	m.xs = append(m.xs, v)
}

func (m *median) Done() any {
	if len(m.xs) == 0 {
		return nil
	}

	sort.Sort(&m.xs)
	e := m.xs[int(math.Floor(float64(len(m.xs))/2))]
	switch e.kind {
	case sqliteInt:
		return e.i
	case sqliteString:
		return e.s
	case sqliteReal:
		return e.r
	case sqliteBlob:
		return e.b
	}

	return nil
}

var aggregateFunctions = map[string]any{
	"stddev":        newStddev,
	"stdev":         newStddev,
	"stddev_pop":    newStddev,
	"mode":          newMode,
	"median":        newMedian,
	"percentile_25": newPercentileN(25),
	"perc_25":       newPercentileN(25),
	"percentile_50": newPercentileN(50),
	"perc_50":       newPercentileN(50),
	"percentile_75": newPercentileN(75),
	"perc_75":       newPercentileN(75),
	"percentile_90": newPercentileN(90),
	"perc_90":       newPercentileN(90),
	"percentile_95": newPercentileN(95),
	"perc_95":       newPercentileN(95),
	"percentile_99": newPercentileN(99),
	"perc_99":       newPercentileN(99),
	"percentile":    newPercentile,
	"perc":          newPercentile,
}
