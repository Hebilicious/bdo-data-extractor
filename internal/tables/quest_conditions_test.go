package tables

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// The Velia dailies exclude each other, so each quest's accept condition names
// its sibling. A decoder that hands a quest the block stored inside its own
// record gives it the next quest's conditions instead.
func TestDecodeQuestConditionsReadsTheBlockBeforeEachRecord(t *testing.T) {
	const (
		looming   = uint32(7<<16 | 1099)
		predators = uint32(8<<16 | 1099)
		hungry    = uint32(10<<16 | 3704)
	)
	list := testQuestList(looming, predators, hungry)

	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, 3)
	data = append(data, testFirstQuestConditionBlock("getLifeLevel(9)>2;!clearQuest(3061,7);", "killmonster(21429,1);", "Sink the Goldmont battleship;")...)
	data = append(data, testQuestRecordBody(looming)...)
	data = append(data, testQuestConditionBlock(looming, "getLifeLevel(9)>2;!clearQuest(3061,8);", "killmonster(21427,3);", "Destroy the small battleship;")...)
	data = append(data, testQuestRecordBody(predators)...)
	data = append(data, testQuestConditionBlock(predators, "getLifeLevel(9)>1;", "killmonster(21415,3);\nkillmonster(21416,3);", "Defeat Hungry Hekaru;")...)
	data = append(data, testQuestRecordBody(hungry)...)

	got, err := DecodeQuestConditions(list, data)
	if err != nil {
		t.Fatal(err)
	}
	want := map[uint32]QuestConditionRow{
		looming:   {AcceptDSL: "getLifeLevel(9)>2;!clearQuest(3061,7);", CompleteDSL: "killmonster(21429,1);", SourceObjective: "Sink the Goldmont battleship;"},
		predators: {AcceptDSL: "getLifeLevel(9)>2;!clearQuest(3061,8);", CompleteDSL: "killmonster(21427,3);", SourceObjective: "Destroy the small battleship;"},
		hungry:    {AcceptDSL: "getLifeLevel(9)>1;", CompleteDSL: "killmonster(21415,3);\nkillmonster(21416,3);", SourceObjective: "Defeat Hungry Hekaru;"},
	}
	for id, w := range want {
		g := got[id]
		if g.AcceptDSL != w.AcceptDSL || g.CompleteDSL != w.CompleteDSL || g.SourceObjective != w.SourceObjective {
			t.Errorf("quest %d-%d conditions = %q / %q / %q, want %q / %q / %q",
				uint16(id), id>>16, g.AcceptDSL, g.CompleteDSL, g.SourceObjective, w.AcceptDSL, w.CompleteDSL, w.SourceObjective)
		}
	}
	if len(got) != len(want) {
		t.Errorf("decoded %d quests, want %d", len(got), len(want))
	}
	if len(got[looming].UnknownHeader) != testFirstQuestConditionHeaderSize || len(got[looming].UnknownObjectivePrefix) != 16 {
		t.Errorf("first block header %d bytes, objective prefix %d bytes", len(got[looming].UnknownHeader), len(got[looming].UnknownObjectivePrefix))
	}
	if len(got[predators].UnknownHeader) != questConditionHeaderSize || len(got[predators].UnknownObjectivePrefix) != 24 || got[predators].UnknownEnd != 3 {
		t.Errorf("later block = %+v", got[predators])
	}
}

func TestDecodeQuestConditionsRejectsAQuestWithoutABlock(t *testing.T) {
	const (
		first  = uint32(1<<16 | 895)
		second = uint32(2<<16 | 895)
	)
	list := testQuestList(first, second)
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, 2)
	data = append(data, testFirstQuestConditionBlock("ClearQuest(4501,3);", "meet(50493,1);", "Ask Hakan;")...)
	data = append(data, testQuestRecordBody(first)...)
	data = append(data, testQuestRecordBody(second)...)

	if _, err := DecodeQuestConditions(list, data); err == nil {
		t.Fatal("decoded a quest whose condition block is missing")
	}
}

const testFirstQuestConditionHeaderSize = 20

func testQuestList(ids ...uint32) []byte {
	records := make([]byte, len(ids)*4)
	for i, id := range ids {
		binary.LittleEndian.PutUint32(records[i*4:], id)
	}
	return testPABRWithUTF16Strings(uint32(len(ids)), records)
}

func testQuestRecordBody(id uint32) []byte {
	var record bytes.Buffer
	binary.Write(&record, binary.LittleEndian, id)
	record.Write(make([]byte, 64))
	return record.Bytes()
}

func testFirstQuestConditionBlock(accept, complete, objective string) []byte {
	var block bytes.Buffer
	header := make([]byte, testFirstQuestConditionHeaderSize)
	header[2] = 1
	block.Write(header)
	writeUTF16(&block, accept)
	writeUTF16(&block, complete)
	block.Write(make([]byte, 16))
	writeUTF16(&block, objective)
	binary.Write(&block, binary.LittleEndian, uint32(11))
	return block.Bytes()
}

func testQuestConditionBlock(previousID uint32, accept, complete, objective string) []byte {
	var block bytes.Buffer
	binary.Write(&block, binary.LittleEndian, previousID)
	block.Write(make([]byte, questConditionHeaderSize))
	writeUTF16(&block, accept)
	writeUTF16(&block, complete)
	block.Write(make([]byte, 24))
	writeUTF16(&block, objective)
	binary.Write(&block, binary.LittleEndian, uint32(3))
	return block.Bytes()
}
