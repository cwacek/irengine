package indexer

import "sort"
import "math"
import log "github.com/cihub/seelog"

type PostingListPruner interface {
	// Prune the posting list <pl> according
	// to some function
	Prune(term LexiconTerm)
}

type DocCountPruner struct {
	Count int
}

type sortable_pl_entries []PostingListEntry

func (s sortable_pl_entries) Len() int {
	return len(s)
}

// Higher numbers come first (higher TF is better)
func (s sortable_pl_entries) Less(i, j int) bool {
	return s[i].Frequency() > s[j].Frequency()
}

func (s sortable_pl_entries) Swap(i, j int) {
	tmp := s[i]
	s[i] = s[j]
	s[j] = tmp
}

func (p *DocCountPruner) Prune(term LexiconTerm) {

	var (
		i     int
		entry PostingListEntry
		pl    PostingList
	)

	pl = term.PostingList()

	// Make a list of PostingListEntries. They're sorted by
	// Document ID, and to remove them we need to sort by
	// TF
	var entries = make(sortable_pl_entries, 0, pl.Len())
	for it := pl.Iterator(); it.Next(); {
		entries = append(entries, it.Value())
	}
	sort.Sort(entries)

	//Remove those we don't want, one by one
	for i, entry = range entries {
		if i >= p.Count {
			pl.Remove(entry.DocId())
		}
	}
}

type TFPruner struct {
	Multiplier float64
}

func (p *TFPruner) Prune(term LexiconTerm) {

	var (
		mean, std_dev float64
		tmp           float64
		threshold     float64
		pl            PostingList
		entry         PostingListEntry
	)

	// Calculate the standard deviation over the
	// posting list frequencies, except don't bother
	// if there's only one thing in the list
	pl = term.PostingList()
	if pl.Len() < 2 {
		return
	}

	mean = float64(term.Tf()) / float64(pl.Len())

	for it := pl.Iterator(); it.Next(); {
		tmp = mean - float64(it.Value().Frequency())

		std_dev += math.Pow(tmp, 2.0)
	}
	std_dev = math.Sqrt(std_dev / float64(pl.Len()))

	//Threshold is mean + <param> * std_dev
	threshold = mean + (p.Multiplier * std_dev)
	log.Infof("Calculated threshold for %s as %0.2f + %0.2f = %0.2f. Removing all PL with lower frequency.",
		term.Text(), mean, std_dev, threshold)

	for it := pl.Iterator(); it.Next(); {
		entry = it.Value()
		if float64(entry.Frequency()) < threshold && pl.Len() > 1 {
			log.Infof("Removed %d from %s because it's TF was %d < %0.2f",
				entry.DocId(), term.Text(), entry.Frequency(), threshold)
			pl.Remove(entry.DocId())
		}
	}
}
