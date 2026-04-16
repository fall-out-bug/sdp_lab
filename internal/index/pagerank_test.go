package index

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPageRank_BasicGraph(t *testing.T) {
	s := openTestStore(t)

	// Create a simple graph: A -> B -> C
	idA, err := s.InsertChunk(Chunk{
		FilePath: "a.go", SymbolName: "A", Kind: "function",
		Language: "go", LineStart: 1, LineEnd: 5, Content: "func A() {}", Hash: "ha",
	})
	require.NoError(t, err)
	idB, err := s.InsertChunk(Chunk{
		FilePath: "b.go", SymbolName: "B", Kind: "function",
		Language: "go", LineStart: 1, LineEnd: 5, Content: "func B() {}", Hash: "hb",
	})
	require.NoError(t, err)
	idC, err := s.InsertChunk(Chunk{
		FilePath: "c.go", SymbolName: "C", Kind: "function",
		Language: "go", LineStart: 1, LineEnd: 5, Content: "func C() {}", Hash: "hc",
	})
	require.NoError(t, err)

	_, err = s.InsertEdge(Edge{SourceID: idA, TargetID: idB, Relation: "calls"})
	require.NoError(t, err)
	_, err = s.InsertEdge(Edge{SourceID: idB, TargetID: idC, Relation: "calls"})
	require.NoError(t, err)

	updated, err := ComputePageRank(s)
	require.NoError(t, err)
	assert.Equal(t, 3, updated)

	// Verify scores were written back
	chunkA, err := s.GetChunk(idA)
	require.NoError(t, err)
	chunkB, err := s.GetChunk(idB)
	require.NoError(t, err)
	chunkC, err := s.GetChunk(idC)
	require.NoError(t, err)

	// C has the highest PageRank because both A and B indirectly point to it
	// A has the lowest because nothing points to it
	assert.Greater(t, chunkC.PageRank, chunkB.PageRank, "C should outrank B")
	assert.Greater(t, chunkB.PageRank, chunkA.PageRank, "B should outrank A")
}

func TestPageRank_HubNode(t *testing.T) {
	s := openTestStore(t)

	// Create a hub: B is pointed to by A and C (B is the most important node)
	idA, err := s.InsertChunk(Chunk{
		FilePath: "a.go", SymbolName: "A", Kind: "function",
		Language: "go", LineStart: 1, LineEnd: 5, Content: "func A() { B() }", Hash: "ha",
	})
	require.NoError(t, err)
	idB, err := s.InsertChunk(Chunk{
		FilePath: "b.go", SymbolName: "B", Kind: "function",
		Language: "go", LineStart: 1, LineEnd: 5, Content: "func B() {}", Hash: "hb",
	})
	require.NoError(t, err)
	idC, err := s.InsertChunk(Chunk{
		FilePath: "c.go", SymbolName: "C", Kind: "function",
		Language: "go", LineStart: 1, LineEnd: 5, Content: "func C() { B() }", Hash: "hc",
	})
	require.NoError(t, err)

	_, err = s.InsertEdge(Edge{SourceID: idA, TargetID: idB, Relation: "calls"})
	require.NoError(t, err)
	_, err = s.InsertEdge(Edge{SourceID: idC, TargetID: idB, Relation: "calls"})
	require.NoError(t, err)

	updated, err := ComputePageRank(s)
	require.NoError(t, err)
	assert.Equal(t, 3, updated)

	chunkA, _ := s.GetChunk(idA)
	chunkB, _ := s.GetChunk(idB)
	chunkC, _ := s.GetChunk(idC)

	// B should have the highest PageRank (two nodes point to it)
	assert.Greater(t, chunkB.PageRank, chunkA.PageRank, "B should outrank A")
	assert.Greater(t, chunkB.PageRank, chunkC.PageRank, "B should outrank C")
	// A and C should be roughly equal
	assert.InDelta(t, chunkA.PageRank, chunkC.PageRank, 0.01, "A and C should have similar scores")
}

func TestPageRank_NoEdges(t *testing.T) {
	s := openTestStore(t)

	// Insert chunks with no edges
	_, err := s.InsertChunk(Chunk{
		FilePath: "a.go", SymbolName: "A", Kind: "function",
		Language: "go", LineStart: 1, LineEnd: 5, Content: "func A() {}", Hash: "ha",
	})
	require.NoError(t, err)

	updated, err := ComputePageRank(s)
	require.NoError(t, err)
	assert.Equal(t, 0, updated, "no edges means no nodes in the graph")
}

func TestPageRank_EmptyStore(t *testing.T) {
	s := openTestStore(t)

	updated, err := ComputePageRank(s)
	require.NoError(t, err)
	assert.Equal(t, 0, updated)
}

func TestPageRank_ScoresSumToOne(t *testing.T) {
	s := openTestStore(t)

	// Create a connected graph
	ids := make([]int64, 5)
	for i := 0; i < 5; i++ {
		id, err := s.InsertChunk(Chunk{
			FilePath: "f.go", SymbolName: "F", Kind: "function",
			Language: "go", LineStart: 1, LineEnd: 5, Content: "func F() {}", Hash: "h",
		})
		require.NoError(t, err)
		ids[i] = id
	}

	// Create a cycle
	for i := 0; i < 5; i++ {
		_, err := s.InsertEdge(Edge{SourceID: ids[i], TargetID: ids[(i+1)%5], Relation: "calls"})
		require.NoError(t, err)
	}

	updated, err := ComputePageRank(s)
	require.NoError(t, err)
	assert.Equal(t, 5, updated)

	// Sum of PageRank scores should be approximately 1.0
	totalPR := 0.0
	for _, id := range ids {
		chunk, err := s.GetChunk(id)
		require.NoError(t, err)
		totalPR += chunk.PageRank
	}
	assert.InDelta(t, 1.0, totalPR, 0.05, "PageRank scores should sum to ~1.0")
}

func TestPageRank_PreservesDirection(t *testing.T) {
	s := openTestStore(t)

	// A star: center is B, A,C,D all point to B
	idB, err := s.InsertChunk(Chunk{
		FilePath: "b.go", SymbolName: "Hub", Kind: "function",
		Language: "go", LineStart: 1, LineEnd: 5, Content: "func Hub() {}", Hash: "hb",
	})
	require.NoError(t, err)

	outerIDs := make([]int64, 3)
	for i := 0; i < 3; i++ {
		id, err := s.InsertChunk(Chunk{
			FilePath: "o.go", SymbolName: "Outer", Kind: "function",
			Language: "go", LineStart: 1, LineEnd: 5, Content: "func Outer() {}", Hash: "ho",
		})
		require.NoError(t, err)
		outerIDs[i] = id

		_, err = s.InsertEdge(Edge{SourceID: id, TargetID: idB, Relation: "calls"})
		require.NoError(t, err)
	}

	_, err = ComputePageRank(s)
	require.NoError(t, err)

	hub, _ := s.GetChunk(idB)
	assert.Greater(t, hub.PageRank, 0.3, "hub node should have high PageRank")
}
