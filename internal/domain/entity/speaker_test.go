package entity_test

import (
	"testing"

	"github.com/canpok1/vox-actor/internal/domain/entity"
)

func TestBuildSpeakerStyleLookup_FlattensStyles(t *testing.T) {
	speakers := []entity.Speaker{
		{
			Name: "ずんだもん",
			Styles: []entity.SpeakerStyle{
				{ID: 3, Name: "ノーマル"},
				{ID: 1, Name: "あまあま"},
			},
		},
		{
			Name: "四国めたん",
			Styles: []entity.SpeakerStyle{
				{ID: 2, Name: "ノーマル"},
			},
		},
	}

	lookup := entity.BuildSpeakerStyleLookup(speakers)

	if got, want := len(lookup), 3; got != want {
		t.Fatalf("len(lookup) = %d, want %d", got, want)
	}

	cases := []struct {
		id      int
		speaker string
		style   string
	}{
		{3, "ずんだもん", "ノーマル"},
		{1, "ずんだもん", "あまあま"},
		{2, "四国めたん", "ノーマル"},
	}
	for _, c := range cases {
		got, ok := lookup[c.id]
		if !ok {
			t.Errorf("lookup[%d] missing", c.id)
			continue
		}
		if got.SpeakerName != c.speaker || got.StyleName != c.style {
			t.Errorf("lookup[%d] = %+v, want speaker=%s style=%s", c.id, got, c.speaker, c.style)
		}
	}
}

func TestBuildSpeakerStyleLookup_DuplicateIDKeepsFirst(t *testing.T) {
	speakers := []entity.Speaker{
		{
			Name:   "話者A",
			Styles: []entity.SpeakerStyle{{ID: 5, Name: "スタイルA"}},
		},
		{
			Name:   "話者B",
			Styles: []entity.SpeakerStyle{{ID: 5, Name: "スタイルB"}},
		},
	}

	lookup := entity.BuildSpeakerStyleLookup(speakers)

	got := lookup[5]
	if got.SpeakerName != "話者A" || got.StyleName != "スタイルA" {
		t.Errorf("lookup[5] = %+v, want speaker=話者A style=スタイルA (first wins)", got)
	}
}

func TestBuildSpeakerStyleLookup_EmptyInput(t *testing.T) {
	lookup := entity.BuildSpeakerStyleLookup(nil)
	if len(lookup) != 0 {
		t.Errorf("lookup should be empty for nil input, got %d entries", len(lookup))
	}
}

func TestBuildSpeakerStyleLookupWithOrder_PresservesOrder(t *testing.T) {
	speakers := []entity.Speaker{
		{
			Name: "ずんだもん",
			Styles: []entity.SpeakerStyle{
				{ID: 3, Name: "ノーマル"},
				{ID: 1, Name: "あまあま"},
			},
		},
		{
			Name: "四国めたん",
			Styles: []entity.SpeakerStyle{
				{ID: 2, Name: "ノーマル"},
			},
		},
	}

	lookup, orderedIDs := entity.BuildSpeakerStyleLookupWithOrder(speakers)

	if got, want := len(lookup), 3; got != want {
		t.Fatalf("len(lookup) = %d, want %d", got, want)
	}

	wantIDs := []int{3, 1, 2}
	if len(orderedIDs) != len(wantIDs) {
		t.Fatalf("len(orderedIDs) = %d, want %d", len(orderedIDs), len(wantIDs))
	}
	for i, want := range wantIDs {
		if got := orderedIDs[i]; got != want {
			t.Errorf("orderedIDs[%d] = %d, want %d", i, got, want)
		}
	}

	cases := []struct {
		id      int
		speaker string
		style   string
	}{
		{3, "ずんだもん", "ノーマル"},
		{1, "ずんだもん", "あまあま"},
		{2, "四国めたん", "ノーマル"},
	}
	for _, c := range cases {
		got, ok := lookup[c.id]
		if !ok {
			t.Errorf("lookup[%d] missing", c.id)
			continue
		}
		if got.SpeakerName != c.speaker || got.StyleName != c.style {
			t.Errorf("lookup[%d] = %+v, want speaker=%s style=%s", c.id, got, c.speaker, c.style)
		}
	}
}

func TestBuildSpeakerStyleLookupWithOrder_DuplicateIDKeepsFirst(t *testing.T) {
	speakers := []entity.Speaker{
		{
			Name:   "話者A",
			Styles: []entity.SpeakerStyle{{ID: 5, Name: "スタイルA"}},
		},
		{
			Name:   "話者B",
			Styles: []entity.SpeakerStyle{{ID: 5, Name: "スタイルB"}},
		},
	}

	lookup, orderedIDs := entity.BuildSpeakerStyleLookupWithOrder(speakers)

	got := lookup[5]
	if got.SpeakerName != "話者A" || got.StyleName != "スタイルA" {
		t.Errorf("lookup[5] = %+v, want speaker=話者A style=スタイルA (first wins)", got)
	}

	wantIDs := []int{5}
	if len(orderedIDs) != len(wantIDs) {
		t.Fatalf("len(orderedIDs) = %d, want %d (duplicate not included twice)", len(orderedIDs), len(wantIDs))
	}
	if orderedIDs[0] != 5 {
		t.Errorf("orderedIDs[0] = %d, want 5", orderedIDs[0])
	}
}

func TestBuildSpeakerStyleLookupWithOrder_EmptyInput(t *testing.T) {
	lookup, orderedIDs := entity.BuildSpeakerStyleLookupWithOrder(nil)
	if len(lookup) != 0 {
		t.Errorf("lookup should be empty for nil input, got %d entries", len(lookup))
	}
	if len(orderedIDs) != 0 {
		t.Errorf("orderedIDs should be empty for nil input, got %d entries", len(orderedIDs))
	}
}
