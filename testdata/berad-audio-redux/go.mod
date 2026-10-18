module github.com/bradhannah/UltimaVRedux/testdata/berad-audio-redux

go 1.27.0

require (
	github.com/bradhannah/UltimaVRedux v0.0.0
	github.com/bradhannah/UltimaVRedux/modules/berad-audio v0.0.0
)

// These substitutions exist only in the staged integration fixture; the gate
// supplies the game path and copies the independent module beside this consumer.
replace github.com/bradhannah/UltimaVRedux/modules/berad-audio => ../library
