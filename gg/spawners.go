package gg

import (
	"log"
	"path"
	//	"runtime/debug"

	"ulambda/fslib"
)

func spawnInputDownloaders(launch ExecutorLauncher, targetHash string, dstDir string, inputs []string, exitDeps []string) []string {
	downloaders := []string{}
	// Make sure to download target thunk file
	downloaders = append(downloaders, spawnDownloader(launch, targetHash, dstDir, "blobs", exitDeps))
	for _, input := range inputs {
		if isThunk(input) {
			uploaderDeps := []string{
				dirUploaderPid(input, GG_REDUCTIONS),
				dirUploaderPid(input, GG_BLOBS),
				outputHandlerPid(input),
			}
			// Download the thunk reduction file as well as the value it points to
			reductionDownloader := spawnDownloader(launch, input, dstDir, "reductions", append(exitDeps, uploaderDeps...))
			// set target == targetReduction to preserve target name
			reductionWriter := spawnReductionWriter(launch, input, input, dstDir, path.Join(".gg", "blobs"), append(exitDeps, uploaderDeps...))
			downloaders = append(downloaders, reductionDownloader, reductionWriter)
		} else {
			downloaders = append(downloaders, spawnDownloader(launch, input, dstDir, "blobs", exitDeps))
		}
	}
	return downloaders
}

// Given a PID, create a no-op which waits on that Pid
func spawnNoOp(launch ExecutorLauncher, waitPid string) string {
	noOpPid := noOpPid(waitPid)
	exitDep := []string{waitPid}
	err := launch.SpawnNoOp(noOpPid, exitDep)
	if err != nil {
		log.Fatalf("Error spawning noop [%v]: %v\n", noOpPid, err)
	}
	return noOpPid
}

func spawnDownloader(launch ExecutorLauncher, targetHash string, dstDir string, subDir string, exitDeps []string) string {
	a := fslib.Attr{}
	a.Pid = downloaderPid(dstDir, subDir, targetHash)
	//	log.Printf("Spawning d %v deps %v", a.Pid, exitDeps)
	//	debug.PrintStack()
	a.Program = "./bin/fsdownloader"
	a.Args = []string{
		ggRemote(subDir, targetHash),
		path.Join(dstDir, ".gg", subDir, targetHash),
	}
	a.Env = []string{}
	a.PairDep = []fslib.PDep{}
	a.ExitDep = exitDeps
	err := launch.Spawn(&a)
	if err != nil {
		log.Fatalf("Error spawning download worker [%v]: %v\n", targetHash, err)
	}
	return a.Pid
}

func spawnDirUploader(launch ExecutorLauncher, targetHash string, subDir string) string {
	a := fslib.Attr{}
	a.Pid = dirUploaderPid(targetHash, subDir)
	a.Program = "./bin/fsdiruploader"
	a.Args = []string{
		ggLocal(targetHash, subDir, ""),
		ggRemote(subDir, ""),
		targetHash,
	}
	a.Env = []string{}
	a.PairDep = []fslib.PDep{}
	a.ExitDep = []string{executorPid(targetHash)} // XXX make this dep explicit
	err := launch.Spawn(&a)
	if err != nil {
		log.Fatalf("Error spawning upload worker [%v]: %v\n", targetHash, err)
	}
	return a.Pid
}

func spawnOrigDirUploader(launch ExecutorLauncher, dir string, subDir string) string {
	a := fslib.Attr{}
	a.Pid = origDirUploaderPid(subDir)
	a.Program = "./bin/fsdiruploader"
	a.Args = []string{
		ggOrig(dir, subDir, ""),
		ggRemote(subDir, ""),
		"",
	}
	a.Env = []string{}
	a.PairDep = []fslib.PDep{}
	a.ExitDep = []string{}
	err := launch.Spawn(&a)
	if err != nil {
		log.Fatalf("Error spawning orig dir upload worker [%v/%v]: %v\n", dir, subDir, err)
	}
	return a.Pid
}

func spawnUploader(launch ExecutorLauncher, targetHash string, srcDir string, subDir string) string {
	a := fslib.Attr{}
	a.Pid = uploaderPid(srcDir, subDir, targetHash)
	a.Program = "./bin/fsuploader"
	a.Args = []string{
		path.Join(srcDir, ".gg", subDir, targetHash),
		ggRemote(subDir, targetHash),
	}
	a.Env = []string{}
	a.PairDep = []fslib.PDep{}
	a.ExitDep = nil
	err := launch.Spawn(&a)
	if err != nil {
		log.Fatalf("Error spawning upload worker [%v]: %v\n", targetHash, err)
	}
	return a.Pid
}

func spawnThunkResultUploaders(launch ExecutorLauncher, hash string) []string {
	uploaders := []string{}
	subDirs := []string{GG_BLOBS, GG_REDUCTIONS, GG_HASH_CACHE}

	// Upload contents of each subdir (blobs, reductions, hash_cache) to 9P remote
	// server
	for _, subDir := range subDirs {
		uploaders = append(uploaders, spawnDirUploader(launch, hash, subDir))
	}
	return uploaders
}

func spawnReductionWriter(launch ExecutorLauncher, target string, targetReduction string, dstDir string, subDir string, deps []string) string {
	a := fslib.Attr{}
	a.Pid = reductionWriterPid(dstDir, subDir, target)
	a.Program = "./bin/gg-target-writer"
	a.Args = []string{
		path.Join(dstDir, subDir),
		target,
		targetReduction,
	}
	a.Env = []string{}
	a.PairDep = []fslib.PDep{}
	reductionPid := outputHandlerPid(targetReduction)
	deps = append(deps, reductionPid)
	a.ExitDep = deps
	err := launch.Spawn(&a)
	if err != nil {
		log.Fatalf("Error spawning target writer [%v]: %v\n", target, err)
	}
	return a.Pid
}

func spawnExecutor(launch ExecutorLauncher, targetHash string, depPids []string) string {
	a := fslib.Attr{}
	a.Pid = executorPid(targetHash)
	a.Program = "./bin/gg-executor"
	a.Args = []string{
		targetHash,
	}
	a.Dir = ""
	a.PairDep = []fslib.PDep{}
	a.ExitDep = depPids
	err := launch.Spawn(&a)
	if err != nil {
		// XXX Clean this up better with caching
		log.Fatalf("Error spawning executor [%v]: %v\n", targetHash, err)
	}
	return a.Pid
}

func spawnThunkOutputHandler(launch ExecutorLauncher, deps []string, thunkHash string, outputFiles []string) string {
	a := fslib.Attr{}
	a.Pid = outputHandlerPid(thunkHash)
	a.Program = "./bin/gg-thunk-output-handler"
	a.Args = []string{
		thunkHash,
	}
	a.Args = append(a.Args, outputFiles...)
	a.Env = []string{}
	a.PairDep = []fslib.PDep{}
	a.ExitDep = deps
	err := launch.Spawn(&a)
	if err != nil {
		// XXX Clean this up better with caching
		//    log.Fatalf("Error spawning output handler [%v]: %v\n", thunkHash, err);
	}
	return a.Pid
}
