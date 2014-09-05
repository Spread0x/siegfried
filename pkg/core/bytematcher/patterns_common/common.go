package patterns_common

import . "github.com/richardlehane/siegfried/pkg/core/bytematcher/patterns"

// Shared test sequences (exported so they can be used by the other bytematcher packages)
var TestSequences = []Sequence{
	Sequence("test"),
	Sequence("test"),
	Sequence("testy"),
	Sequence("TEST"),
	Sequence("TESTY"),
	Sequence("YNESS"), //5
	Sequence{'a'},
	Sequence{'b'},
	Sequence{'c'},
	Sequence{'d'},
	Sequence{'e'},
	Sequence{'f'},
	Sequence{'g'},
	Sequence{'h'},
	Sequence{'i'},
	Sequence{'j'},
	Sequence("junk"), // 16
	Sequence("23"),
}

// Shared test choices (exported so they can be used by the other bytematcher packages)
var TestChoices = []Choice{
	Choice{TestSequences[0], TestSequences[2]},
	Choice{TestSequences[2], TestSequences[0]},
	Choice{TestSequences[4], TestSequences[5]},
	Choice{TestSequences[3]},
	Choice{
		TestSequences[6],
		TestSequences[7],
		TestSequences[8],
		TestSequences[9],
		TestSequences[10],
		TestSequences[11],
		TestSequences[12],
		TestSequences[13],
		TestSequences[14],
		TestSequences[15],
	},
}
