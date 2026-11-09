package acd

import (
	"sort"

	"github.com/scc-platform/shared/fn"
)

func sortAgentsBy(agents []Agent) []Agent {
	sort.SliceStable(agents, func(i, j int) bool {
		left, right := agents[i], agents[j]
		return left.Handled < right.Handled || (left.Handled == right.Handled && left.LastCallAt.Before(right.LastCallAt))
	})
	return agents
}

func (a Agent) edition(ext string) Agent {
	return fn.With(a, func(next *Agent) { next.Ext = ext })
}

func DefaultAgents() []Agent {
	return []Agent{
		{Ext: "101", Name: "Анна Кузнецова", Team: "приёмная комиссия", Skills: []string{"admission", "general"}, State: "ready"},
		{Ext: "104", Name: "Игорь Лебедев", Team: "приёмная комиссия", Skills: []string{"admission"}, State: "ready"},
		{Ext: "107", Name: "Мария Соколова", Team: "поддержка", Skills: []string{"support", "general"}, State: "ready"},
		{Ext: "112", Name: "Пётр Васильев", Team: "поддержка", Skills: []string{"support"}, State: "paused"},
		{Ext: "118", Name: "Ольга Морозова", Team: "экзамены", Skills: []string{"exams", "support"}, State: "ready"},
		{Ext: "121", Name: "Дмитрий Орлов", Team: "экзамены", Skills: []string{"exams"}, State: "ready"},
		{Ext: "126", Name: "Елена Новикова", Team: "обзвон", Skills: []string{"callback"}, State: "ready"},
		{Ext: "133", Name: "Сергей Титов", Team: "обзвон", Skills: []string{"callback", "general"}, State: "ready"},
	}
}

func DefaultQueues() map[string][]string {
	return map[string][]string{
		"admission": {"101", "104"},
		"support":   {"107", "112", "118"},
		"exams":     {"118", "121"},
		"callback":  {"126", "133"},
	}
}
