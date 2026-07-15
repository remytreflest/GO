package embedding

import (
	"math"
	"strings"
	"testing"
)

func norm(v []float32) float64 {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	return math.Sqrt(sum)
}

func TestEmbed_Deterministic(t *testing.T) {
	a := Embed("goroutines and channels")
	b := Embed("goroutines and channels")
	if len(a) != Dimension {
		t.Fatalf("expected dimension %d, got %d", Dimension, len(a))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("Embed must be deterministic, differed at index %d: %v vs %v", i, a[i], b[i])
		}
	}
}

func TestEmbed_Normalized(t *testing.T) {
	v := Embed("some arbitrary note content about postgres and vectors")
	n := norm(v)
	if math.Abs(n-1) > 1e-6 {
		t.Fatalf("expected unit norm, got %f", n)
	}
}

func TestEmbed_EmptyIsNotZeroVector(t *testing.T) {
	v := Embed("")
	n := norm(v)
	if n == 0 {
		t.Fatalf("Embed(\"\") must not be the zero vector")
	}
	if math.Abs(n-1) > 1e-6 {
		t.Fatalf("expected unit norm for empty input, got %f", n)
	}
}

func TestEmbed_SimilarTextsCluster(t *testing.T) {
	a := Embed("goroutines channels worker pool")
	b := Embed("goroutines channels worker pool timeout")
	c := Embed("recette de pates a la carbonara")

	cosine := func(x, y []float32) float64 {
		var dot float64
		for i := range x {
			dot += float64(x[i]) * float64(y[i])
		}
		return dot
	}

	simAB := cosine(a, b)
	simAC := cosine(a, c)
	if simAB <= simAC {
		t.Fatalf("expected lexically similar texts to be closer: sim(a,b)=%f sim(a,c)=%f", simAB, simAC)
	}
}

func TestExtractTags(t *testing.T) {
	text := "Go goroutines goroutines channels channels channels worker pool"
	tags := ExtractTags(text, 3)
	if len(tags) != 3 {
		t.Fatalf("expected 3 tags, got %v", tags)
	}
	if tags[0] != "channels" {
		t.Fatalf("expected most frequent token first, got %v", tags)
	}
}

func TestExtractTags_StopwordsExcluded(t *testing.T) {
	tags := ExtractTags("the go and the api for the developer", 10)
	for _, tag := range tags {
		if tag == "the" || tag == "and" || tag == "for" {
			t.Fatalf("stopword %q leaked into tags: %v", tag, tags)
		}
	}
}

func TestExtractTags_MaxCappedByAvailable(t *testing.T) {
	tags := ExtractTags("golang", 5)
	if len(tags) != 1 {
		t.Fatalf("expected 1 tag when only 1 candidate exists, got %v", tags)
	}
}

func TestSummarize_ShortSentenceKept(t *testing.T) {
	got := Summarize("Short note. More detail follows here.", 50)
	if got != "Short note." {
		t.Fatalf("expected first sentence, got %q", got)
	}
}

func TestSummarize_HardTruncationAtWordBoundary(t *testing.T) {
	text := "one two three four five six seven eight nine ten"
	got := Summarize(text, 15)
	if len(got) == 0 {
		t.Fatalf("expected non-empty summary")
	}
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("expected truncated summary to end with ellipsis, got %q", got)
	}
}

func TestSummarize_Empty(t *testing.T) {
	if got := Summarize("   ", 50); got != "" {
		t.Fatalf("expected empty summary for blank input, got %q", got)
	}
}

func TestScore_Bounds(t *testing.T) {
	if s := Score("", ""); s != 0 {
		t.Fatalf("expected 0 score for empty note, got %f", s)
	}
	long := ""
	for i := 0; i < 2000; i++ {
		long += "x"
	}
	if s := Score("title", long); s < 0 || s > 1 {
		t.Fatalf("expected score in [0,1], got %f", s)
	}
}

func TestScore_Deterministic(t *testing.T) {
	a := Score("Go", "goroutines and channels")
	b := Score("Go", "goroutines and channels")
	if a != b {
		t.Fatalf("Score must be deterministic: %f vs %f", a, b)
	}
}
