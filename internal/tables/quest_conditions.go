package tables

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/idevelopthings/bdo-data-extractor/internal/bss"
)

const (
	questConditionHeaderSize = 25
	questRowCountSize        = 4
)

// QuestConditionRow is the client-evaluated acceptance and completion DSL for
// one quest record.
type QuestConditionRow struct {
	AcceptDSL              string
	CompleteDSL            string
	SourceObjective        string
	UnknownHeader          []byte
	UnknownObjectivePrefix []byte
	UnknownEnd             uint32
}

// DecodeQuestConditions reads the condition block that precedes each quest.dbss
// record. allquestlist.bss supplies the records' physical order because
// quest.dbss has no separate offset index.
//
// A quest's condition block ends exactly where its record begins: the first
// quest's block sits between the row count and the first record, and every
// later quest's block closes the record before it.
func DecodeQuestConditions(questListData, questData []byte) (map[uint32]QuestConditionRow, error) {
	questIDs, count, err := decodeOrderedQuestIDs(questListData)
	if err != nil {
		return nil, err
	}
	if len(questData) < questRowCountSize || int(bss.U32(questData, 0)) != count {
		return nil, fmt.Errorf("quest: row count does not match allquestlist count %d", count)
	}

	starts := make([]int, len(questIDs))
	from := questRowCountSize
	for i, id := range questIDs {
		pos := indexQuestID(questData, from, id)
		if pos < 0 {
			return nil, fmt.Errorf("quest: ordered quest %d-%d is absent after offset %d", uint16(id), uint16(id>>16), from)
		}
		starts[i] = pos
		from = pos + 4
	}

	rows := make(map[uint32]QuestConditionRow, len(questIDs))
	for i, id := range questIDs {
		var (
			row QuestConditionRow
			ok  bool
		)
		if i == 0 {
			row, ok = findFirstQuestConditionBlock(questData[:starts[0]])
		} else {
			row, ok = findQuestConditionBlock(questData[starts[i-1]:starts[i]], questIDs[i-1])
		}
		if !ok {
			return nil, fmt.Errorf("quest: quest %d-%d has no condition block before its record", uint16(id), uint16(id>>16))
		}
		rows[id] = row
	}
	return rows, nil
}

func decodeOrderedQuestIDs(data []byte) ([]uint32, int, error) {
	pabr, err := bss.OpenPABR(data)
	if err != nil {
		return nil, 0, fmt.Errorf("allquestlist: %w", err)
	}
	recordSize, fixed := pabr.RecordSize()
	if !fixed || recordSize != 4 {
		return nil, 0, fmt.Errorf("allquestlist: record size %d, want 4", recordSize)
	}
	ids := make([]uint32, pabr.Rows)
	seen := make(map[uint32]bool, pabr.Rows)
	for row := range pabr.Rows {
		id := bss.U32(data, pabr.RecordsStart+row*recordSize)
		if id == 0 || seen[id] {
			return nil, 0, fmt.Errorf("allquestlist: invalid or duplicate quest id %08x", id)
		}
		seen[id] = true
		ids[row] = id
	}
	return ids, pabr.Rows, nil
}

func indexQuestID(data []byte, from int, id uint32) int {
	if from < 0 || from > len(data) {
		return -1
	}
	var raw [4]byte
	binary.LittleEndian.PutUint32(raw[:], id)
	rel := bytes.Index(data[from:], raw[:])
	if rel < 0 {
		return -1
	}
	return from + rel
}

// findFirstQuestConditionBlock reads the first quest's condition block, which
// sits between the row count and the first record. It has no preceding record
// whose id could open it, and its header is shorter than the later blocks', so
// its condition strings are found by where they parse through to the record.
func findFirstQuestConditionBlock(prefix []byte) (QuestConditionRow, bool) {
	for stringsPos := questRowCountSize; stringsPos+8 <= len(prefix); stringsPos++ {
		if bss.PeekUTF16Chars(prefix, stringsPos) == 0 {
			continue
		}
		if row, ok := decodeQuestConditionStrings(prefix, stringsPos); ok {
			row.UnknownHeader = bytes.Clone(prefix[questRowCountSize:stringsPos])
			return row, true
		}
	}
	return QuestConditionRow{}, false
}

// findQuestConditionBlock finds the one condition block that closes the
// previous quest's record. The block opens with that previous quest's id and a
// fixed header, so the search skips the record's own leading id at offset 0.
func findQuestConditionBlock(previousRecord []byte, previousID uint32) (QuestConditionRow, bool) {
	var found QuestConditionRow
	matches := 0
	for pos := 1; pos+4 <= len(previousRecord); pos++ {
		if bss.U32(previousRecord, pos) != previousID {
			continue
		}
		headerStart := pos + 4
		stringsPos := headerStart + questConditionHeaderSize
		if row, ok := decodeQuestConditionStrings(previousRecord, stringsPos); ok {
			row.UnknownHeader = bytes.Clone(previousRecord[headerStart:stringsPos])
			found = row
			matches++
		}
	}
	return found, matches == 1
}

// decodeQuestConditionStrings reads the accept and complete DSL at pos, then
// the source objective and final scalar, which must end exactly at the end of
// data, where the owning record begins.
func decodeQuestConditionStrings(data []byte, pos int) (QuestConditionRow, bool) {
	c := bss.NewCursor(data, pos, len(data))
	accept := c.UTF16()
	complete := c.UTF16()
	if !c.OK() {
		return QuestConditionRow{}, false
	}
	conditionEnd := c.Pos()
	for objectivePos := conditionEnd; objectivePos+12 <= len(data); objectivePos++ {
		if bss.PeekUTF16Chars(data, objectivePos) == 0 {
			continue
		}
		end := bss.NewCursor(data, objectivePos, len(data))
		objective := end.UTF16()
		unknownEnd := end.U32()
		if !end.OK() || end.Remaining() != 0 {
			continue
		}
		return QuestConditionRow{
			AcceptDSL:              accept,
			CompleteDSL:            complete,
			SourceObjective:        objective,
			UnknownObjectivePrefix: bytes.Clone(data[conditionEnd:objectivePos]),
			UnknownEnd:             unknownEnd,
		}, true
	}
	return QuestConditionRow{}, false
}
