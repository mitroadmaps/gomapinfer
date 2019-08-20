package common

import (
	"fmt"
	"math"
)

// This is an improved version of Viterbi map matching, where we handle sparse traces by
//  applying multiple transitions based on the distance between observations. For example,
//  if two consecutive samples are k*VITERBI2_GRANULARITY apart, then we will apply (k-1)
//  transitions prior to a transition+emission pair.
// Additionally, Viterbi2 takes an edgeWeights map so that some edges being more likely
//  than other edges can be taken into account in the model. If edgeWeights is nil, then
//  the weights are determined based on the angle between the segments, similar to the
//  process in the old Viterbi function.

const VITERBI2_GRANULARITY = 50
const VITERBI2_SIGMA = 30
const VITERBI2_START_TOLERANCE = 100
const VITERBI2_THREADS = 36

const VITERBI2_MODE = "new"

type Viterbi2Options struct {
	Granularity float64
	Sigma float64
	StartTolerance float64
	Threads int

	// only compute edgeHits, do not store the map-matched data
	HitsOnly bool

	// a map from edge IDs to weight of the ID. If nil, the transition probabilities
	// are weighted based on the angle difference between the source and destination edges.
	EdgeWeights map[int]float64

	// use constant 0.5 weight if not at junction, and 0.05 if at junction
	// and also don't allow u-turn
	NewMode bool

	Output map[int][]EdgePos
}

func (opts Viterbi2Options) GetGranularity() float64 {
	if opts.Granularity != 0 {
		return opts.Granularity
	} else {
		return VITERBI2_GRANULARITY
	}
}

func (opts Viterbi2Options) GetSigma() float64 {
	if opts.Sigma != 0 {
		return opts.Sigma
	} else {
		return VITERBI2_SIGMA
	}
}

func (opts Viterbi2Options) GetStartTolerance() float64 {
	if opts.StartTolerance != 0 {
		return opts.StartTolerance
	} else {
		return VITERBI2_START_TOLERANCE
	}
}

func (opts Viterbi2Options) GetThreads() int {
	if opts.Threads != 0 {
		return opts.Threads
	} else {
		return VITERBI2_THREADS
	}
}

// Map match each trace in traces to the road network specified by graph.
// Returns edgeHits, a map from edge ID to the number of times the edge is passed by a trace.
func Viterbi2(traces []*Trace, graph *Graph, opts Viterbi2Options) (edgeHits map[int]int) {
	// precompute transition probabilities
	transitionProbs := make([]map[int]float64, len(graph.Edges))
	for _, edge := range graph.Edges {
		probs := make(map[int]float64)
		transitionProbs[edge.ID] = probs

		var adjacentEdges []*Edge
		for _, other := range edge.Dst.Out {
			if opts.NewMode && other.Dst == edge.Src {
				continue
			}
			adjacentEdges = append(adjacentEdges, other)
		}

		// on all edges there is 0.5 self loop
		probs[edge.ID] = 0.5
		var totalProb float64 = 0.5

		// compute weights to adjacent edges if needed
		weights := opts.EdgeWeights
		if weights == nil {
			weights = make(map[int]float64)
			for _, other := range adjacentEdges {
				if opts.NewMode {
					if len(adjacentEdges) == 1 {
						weights[other.ID] = 0.5
						continue
					} else {
						weights[other.ID] = 0.05
						continue
					}
				} else {
					negAngle := math.Pi / 2 - edge.AngleTo(other)
					if negAngle < 0 {
						negAngle = 0
					}
					weights[other.ID] = negAngle * negAngle + 0.05
				}
			}
		}

		// extract probabilities from the weights
		// we force the average probability to be at most 0.05
		// any additional probability mass is discarded (essentially it is directed to an
		//  impossible state)
		// NOTE: none of this is used for 'new' mode!
		var totalWeight float64 = 0
		for _, other := range adjacentEdges {
			totalWeight += weights[other.ID]
		}
		averageWeight := totalWeight / float64(len(adjacentEdges))
		averageProb := 0.05
		if averageProb * float64(len(adjacentEdges)) + totalProb > 0.9 {
			averageProb = (0.9 - totalProb) / float64(len(adjacentEdges))
		}

		for _, other := range adjacentEdges {
			var prob float64
			if opts.NewMode {
				prob = weights[other.ID]
			} else {
				prob = averageProb * weights[other.ID] / averageWeight
			}
			probs[other.ID] = prob
			totalProb += prob
		}
	}

	rtree := graph.Rtree()

	// get conditional emission probabilities
	sigma := opts.GetSigma()
	emissionProbs := func(point Point, tolerance float64) map[int]float64 {
		candidates := rtree.Search(point.RectangleTol(tolerance))
		if len(candidates) == 0 {
			return nil
		}
		scores := make(map[int]float64)
		var totalScore float64 = 0
		for _, edge := range candidates {
			distance := edge.Segment().Distance(point)
			score := math.Exp(-0.5 * distance * distance / sigma / sigma)
			scores[edge.ID] = score
			totalScore += score
		}
		for i := range scores {
			scores[i] /= totalScore
		}
		return scores
	}

	// match a single trace
	granularity := opts.GetGranularity()
	startTolerance := opts.GetStartTolerance()
	matchTrace := func(traceIdx int, trace *Trace, edgeHits map[int]int, output map[int][]EdgePos) {
		if len(trace.Observations) < 5 {
			//fmt.Printf("viterbi: warning: too few observations, skipping trace (%d)", len(trace.Observations))
			return
		}
		// initial probability is uniform across candidates
		probs := make(map[int]float64)
		for _, edge := range rtree.Search(trace.Observations[0].Point.RectangleTol(startTolerance)) {
			probs[edge.ID] = 0
		}
		backpointers := make([][]map[int]int, len(trace.Observations))
		for i := 1; i < len(trace.Observations); i++ {
			obs := trace.Observations[i]

			// apply extra transitions in case the vehicle traveled a large distance from the
			//  previous observation
			distance := obs.Point.Distance(trace.Observations[i - 1].Point)
			for distance > granularity && granularity > 0 {
				nextProbs := make(map[int]float64)
				nextBackpointers := make(map[int]int)
				for prevEdgeID := range probs {
					transitions := transitionProbs[prevEdgeID]
					for nextEdgeID := range transitions {
						prob := probs[prevEdgeID] + math.Log(transitions[nextEdgeID])
						if curProb, ok := nextProbs[nextEdgeID]; !ok || prob > curProb {
							nextProbs[nextEdgeID] = prob
							nextBackpointers[nextEdgeID] = prevEdgeID
						}
					}
				}
				backpointers[i] = append(backpointers[i], nextBackpointers)
				probs = nextProbs
				distance -= granularity
			}

			var nextProbs map[int]float64
			var nextBackpointers map[int]int

			// find the most likely to match the emission+transition
			// we use an increasing factor in case there are no edges within a reasonable distance
			//  from the observed point
			for factor := float64(1); len(nextProbs) < 2 && factor <= 4; factor *= 2 {
				nextProbs = make(map[int]float64)
				nextBackpointers = make(map[int]int)
				emissions := emissionProbs(obs.Point, startTolerance * factor)
				if factor > 1 {
					//fmt.Printf("viterbi: warning: factor=%f at i=%d, point=%v\n", factor, i, obs.Point)
				}
				for prevEdgeID := range probs {
					transitions := transitionProbs[prevEdgeID]
					for nextEdgeID := range transitions {
						if emissions[nextEdgeID] == 0 {
							continue
						}
						prob := probs[prevEdgeID] + math.Log(transitions[nextEdgeID]) + math.Log(emissions[nextEdgeID])
						if curProb, ok := nextProbs[nextEdgeID]; !ok || prob > curProb {
							nextProbs[nextEdgeID] = prob
							nextBackpointers[nextEdgeID] = prevEdgeID
						}
					}
				}
			}
			backpointers[i] = append(backpointers[i], nextBackpointers)
			if len(nextProbs) == 0 {
				//fmt.Printf("viterbi: warning: failed to find edge, skipping trace: i=%d, point=%v\n", i, obs.Point)
				return
			}
			probs = nextProbs
		}

		// collect state sequence and annotate trace with map matched data
		var bestEdgeID *int
		for edgeID := range probs {
			if bestEdgeID == nil || probs[edgeID] > probs[*bestEdgeID] {
				bestEdgeID = new(int)
				*bestEdgeID = edgeID
			}
		}
		curEdge := *bestEdgeID
		var edgePosList []EdgePos
		for i := len(trace.Observations) - 1; i >= 0; i-- {
			if !opts.HitsOnly {
				edge := graph.Edges[curEdge]
				position := edge.Segment().Project(trace.Observations[i].Point, false)
				edgePos := EdgePos{edge, position}
				if opts.Output == nil {
					trace.Observations[i].SetMetadata("viterbi", edgePos)
				} else {
					edgePosList = append(edgePosList, edgePos)
				}
			}

			for j := len(backpointers[i]) - 1; j >= 0; j-- {
				prevEdge := backpointers[i][j][curEdge]
				if prevEdge != curEdge {
					edgeHits[curEdge]++
					curEdge = prevEdge
				}
			}
		}
		if output != nil {
			output[traceIdx] = make([]EdgePos, len(edgePosList))
			for i, edgePos := range edgePosList {
				output[traceIdx][len(edgePosList)-i-1] = edgePos
			}
		}
	}

	type traceWithIdx struct {
		idx int
		trace *Trace
	}

	type result struct {
		edgeHits map[int]int
		output map[int][]EdgePos
	}

	traceCh := make(chan traceWithIdx)
	nthreads := opts.GetThreads()
	doneCh := make(chan result)
	for i := 0; i < nthreads; i++ {
		go func() {
			edgeHits := make(map[int]int)
			var output map[int][]EdgePos
			if opts.Output != nil {
				output = make(map[int][]EdgePos)
			}
			for trace := range traceCh {
				matchTrace(trace.idx, trace.trace, edgeHits, output)

			}
			doneCh <- result{edgeHits, output}
		}()
	}
	for traceIdx, trace := range traces {
		if traceIdx % 100 == 0 {
			fmt.Printf("progress: %d/%d\n", traceIdx, len(traces))
		}
		traceCh <- traceWithIdx{traceIdx, trace}
	}
	close(traceCh)
	edgeHits = make(map[int]int)
	for i := 0; i < nthreads; i++ {
		result := <- doneCh
		for edgeID, hits := range result.edgeHits {
			edgeHits[edgeID] += hits
		}
		if opts.Output != nil {
			for traceIdx, edgePosList := range result.output {
				opts.Output[traceIdx] = edgePosList
			}
		}
	}
	return
}
