package analytics

import (
	"math/rand"
	"time"
)

var seedQueues = []string{"admission", "support", "callback", "exams"}
var seedAgents = []string{"101", "104", "107", "112", "118", "121", "126", "133"}

const seedValue = 20261108

func SeedCalls(count int, day time.Time) []Call {
	source := rand.New(rand.NewSource(seedValue))
	return seedInto(make([]Call, 0, count), source, day)
}

func seedInto(calls []Call, source *rand.Rand, day time.Time) []Call {
	for len(calls) < cap(calls) {
		calls = append(calls, buildCall(source, day, len(calls)))
	}
	return calls
}

func buildCall(source *rand.Rand, day time.Time, index int) Call {
	wait := float64(source.Intn(45)) + source.Float64()
	return callOf(source, day, index, wait, wait < 32)
}

func callOf(source *rand.Rand, day time.Time, index int, wait float64, answered bool) Call {
	return Call{
		Queue:       seedQueues[index%len(seedQueues)],
		Agent:       seedAgents[index%len(seedAgents)],
		StartedAt:   day.Add(time.Duration(8*60+index%600) * time.Minute),
		WaitSeconds: wait,
		TalkSeconds: answeredSeconds(source, answered),
		Answered:    answered,
	}
}

func answeredSeconds(source *rand.Rand, answered bool) float64 {
	if answered {
		return float64(source.Intn(420)) + source.Float64()
	}
	return 0
}
