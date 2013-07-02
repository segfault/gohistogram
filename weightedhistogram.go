package gohistogram

// A WeightedHistogram implements Histogram. A WeightedHistogram has bins that have values
// which are exponentially weighted moving averages. This allows you keep inserting large
// amounts of data into the histogram and approximate quantiles with recency factored in.
type WeightedHistogram struct {
	bins    []bin
	maxbins int
	total   float64
	alpha   float64
}

// NewHistogram returns a new NumericHistogram with a maximum of n bins with a decay factor of alpha.
func NewWeightedHistogram(n int, alpha float64) *WeightedHistogram {
	return &WeightedHistogram{
		bins:    make([]bin, 0),
		maxbins: n,
		total:   0,
		alpha:   alpha,
	}
}

func ewma(existingVal float64, newVal float64, alpha float64) (result float64) {
	result = newVal*(1-alpha) + existingVal*alpha
	return
}

func (h *WeightedHistogram) scaleDown(except int) {
	for i := range h.bins {
		if i != except {
			h.bins[i].value = ewma(h.bins[i].value, 0, h.alpha)
		}
	}
}

func (h *WeightedHistogram) Add(n float64) {
	defer h.trim()
	for i := range h.bins {
		if h.bins[i].value == n {
			h.bins[i].count++

			defer h.scaleDown(i)
			return
		}

		if h.bins[i].value > n {

			newbin := bin{value: n, count: 1}
			head := append(make([]bin, 0), h.bins[0:i]...)

			head = append(head, newbin)
			tail := h.bins[i:]
			h.bins = append(head, tail...)

			defer h.scaleDown(i)
			return
		}
	}

	h.bins = append(h.bins, bin{count: 1, value: n})
}

func (h *WeightedHistogram) Quantile(q float64) float64 {
	count := q * float64(h.total)
	for i := range h.bins {
		count -= float64(h.bins[i].count)

		if count <= 0 {
			return h.bins[i].value
		}
	}

	return -1
}

func (h *WeightedHistogram) trim() {
	total := 0.0
	for i := range h.bins {
		total += h.bins[i].count
	}
	h.total = total
	for len(h.bins) > h.maxbins {

		// Find closest bins in terms of value
		minDelta := 1e99
		minDeltaIndex := 0
		for i := range h.bins {
			if i == 0 {
				continue
			}

			if delta := h.bins[i].value - h.bins[i-1].value; delta < minDelta {
				minDelta = delta
				minDeltaIndex = i
			}
		}

		// We need to merge bins minDeltaIndex-1 and minDeltaIndex
		mergedbin := bin{
			value: (h.bins[minDeltaIndex-1].value + h.bins[minDeltaIndex].value) / 2, // average value
			count: h.bins[minDeltaIndex-1].count + h.bins[minDeltaIndex].count,       // summed heights
		}
		head := append(make([]bin, 0), h.bins[0:minDeltaIndex-1]...)
		tail := append([]bin{mergedbin}, h.bins[minDeltaIndex+1:]...)
		h.bins = append(head, tail...)
	}
}