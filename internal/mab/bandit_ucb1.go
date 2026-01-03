package mab

import (
	"math"
	"sync"
)

// NOTE: Since nomenclature may be confusing: 'ARM' is the architecture, 'arm' is the arm of the Multi-Armed Bandit (MAB)

// ArmStats maintains information about a single arm dedicated to a single function
type ArmStats struct {
	Count      int64   // UCB needs to know hom many times we chose that arm/architecture
	SumRewards float64 // Sum of rewards
	AvgReward  float64 // Avg Reward (Q value in the formula)
}

// UCB1Bandit is the bandit that handles decision for ONE function
type UCB1Bandit struct {
	TotalCounts int64                // number of total executions (t)
	Arms        map[string]*ArmStats // Map "amd64" -> Stats, "arm64" -> Stats for each arm
	mu          sync.RWMutex         // Mutex per thread-safety
}

// BanditManager contains all the existing bandits (one for each known function)
type BanditManager struct {
	bandits map[string]*UCB1Bandit
	mu      sync.RWMutex
}

var GlobalBanditManager *BanditManager

// InitBanditManager sets up the bandit manager
func InitBanditManager() {
	GlobalBanditManager = &BanditManager{
		bandits: make(map[string]*UCB1Bandit),
	}
}

// GetBandit returns (or creates) the bandit for a given function
func (bm *BanditManager) GetBandit(functionName string) *UCB1Bandit {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if _, exists := bm.bandits[functionName]; !exists {
		// if we don't have one, then create a new bandit for this function and put it in the bandits map
		bm.bandits[functionName] = &UCB1Bandit{
			TotalCounts: 0,
			Arms: map[string]*ArmStats{
				"amd64": {Count: 0, SumRewards: 0, AvgReward: 0},
				"arm64": {Count: 0, SumRewards: 0, AvgReward: 0},
			},
		}
	}
	return bm.bandits[functionName]
}

// SelectArm implements UCB-1 formulas
// Returns the suggested architecture to use ("amd64" o "arm64")
func (b *UCB1Bandit) SelectArm() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	// 1. If an arm has never been tried, it has absolute priority (Forced Exploration)
	for arch, stats := range b.Arms {
		if stats.Count == 0 {
			return arch
		}
	}

	bestScore := -1.0 // Initialize with a very low score
	bestArch := ""

	// Exploration parameter C (usually sqrt(2) ~= 1.41, but can be tuned)
	// Higher values lead to more exploration. Lower values lead to more exploitation.
	//c := 1.41

	c := 3.0

	// 2. Calculate UCB1 score for each architecture
	for arch, stats := range b.Arms {
		// Formula: Q(a) + c * sqrt( ln(t) / N(a) ) where Q(a) is AvgReward, t is TotalCounts, N(a) is stats.Count
		explorationBonus := c * math.Sqrt(math.Log(float64(b.TotalCounts))/float64(stats.Count))
		score := stats.AvgReward + explorationBonus

		if score > bestScore {
			bestScore = score // Update best score
			bestArch = arch
		}
	}

	return bestArch
}

// UpdateReward updates bandit stats after execution. For now reward is 1.0 / executionTime (not considering setup time).
// It may be fine-tuned in the future.
func (b *UCB1Bandit) UpdateReward(arch string, reward float64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.Arms[arch]; !ok {
		return // Should not happen
	}

	stats := b.Arms[arch]

	// Update global and local counts
	b.TotalCounts++
	stats.Count++

	// Update average reward
	stats.SumRewards += reward
	stats.AvgReward = stats.SumRewards / float64(stats.Count)
}
