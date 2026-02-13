package aggregration

import (
	"context"
	"log-ingestion/internal/types"
)

func Aggregate(ctx context.Context, logsCountChannel <-chan types.LogCounts) types.LogCounts {
	totalLogCounts := make(types.LogCounts)

	for {
		select {
		case <-ctx.Done():
			return totalLogCounts
		case lc, ok := <-logsCountChannel:
			if !ok {
				// channel closed
				return totalLogCounts
			}
			merge(&totalLogCounts, lc)
		}
	}
}

// merge current log count into total log counts
func merge(lc *types.LogCounts, clc types.LogCounts) {
	for outerKey, innerMap := range clc {
		if _, exists := (*lc)[outerKey]; !exists {
			(*lc)[outerKey] = make(map[string]int)
		}
		for innerKey, count := range innerMap {
			(*lc)[outerKey][innerKey] += count
		}
	}
}
