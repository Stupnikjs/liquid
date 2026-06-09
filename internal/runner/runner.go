package runner

import (
	"fmt"
	"strings"
)

func (r *Runner) GetStats() string {
	var sb strings.Builder
	for _, id := range r.MarketConsumer.Cache.Markets.Ids() {
		fmt.Fprintf(&sb, "%s \n", r.MarketConsumer.Cache.Markets.GetSnapshot(id).Analysis().String())

	}
	return sb.String()
}
