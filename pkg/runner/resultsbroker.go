package runner

import (
	"sync"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/logme"
)

type resultsBroker struct {
	mux         sync.RWMutex
	subscribers map[*analysis.Analyzer][]chan any
}

func newResultsBroker() *resultsBroker {
	return &resultsBroker{
		subscribers: make(map[*analysis.Analyzer][]chan any),
	}
}

func (r *resultsBroker) publish(analyzer *analysis.Analyzer, result any) {
	r.mux.RLock()
	defer r.mux.RUnlock()
	logme.DebugFln("publishing result for analyzer %q to %d subscribers", analyzer.Name, len(r.subscribers[analyzer]))
	for _, sub := range r.subscribers[analyzer] {
		sub <- result
	}
}

func (r *resultsBroker) subscribe(analyzer *analysis.Analyzer) <-chan any {
	r.mux.Lock()
	defer r.mux.Unlock()
	ch := make(chan any)
	if _, ok := r.subscribers[analyzer]; !ok {
		r.subscribers[analyzer] = []chan any{}
	}
	r.subscribers[analyzer] = append(r.subscribers[analyzer], ch)
	return ch
}
