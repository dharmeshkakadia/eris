package dappmanager

import (
	// "path/filepath"
	// "bytes"
	// "crypto/sha1"
	// "encoding/hex"
	"encoding/json"
	"errors"
	"github.com/eris-ltd/decerver/interfaces/dapps"
	"github.com/eris-ltd/decerver/interfaces/decerver"
	"github.com/eris-ltd/decerver/interfaces/files"
	"github.com/eris-ltd/decerver/interfaces/logging"
	"github.com/eris-ltd/decerver/interfaces/modules"
	"github.com/eris-ltd/decerver/interfaces/network"
	"github.com/eris-ltd/decerver/interfaces/scripting"
	"github.com/eris-ltd/epm-go/chains"
	"github.com/eris-ltd/epm-go/utils"
	// "github.com/syndtr/goleveldb/leveldb"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	//"time"
	"github.com/robertkrimen/otto/parser"
)

var logger *log.Logger = logging.NewLogger("Dapp Manager")

// const REG_URL = "http://localhost:9999"

type Dapp struct {
	models      []string
	path        string
	packageFile *dapps.PackageFile
}

func (dapp *Dapp) Models() []string {
	return dapp.models
}

func (dapp *Dapp) Path() string {
	return dapp.path
}

func (dapp *Dapp) PackageFile() *dapps.PackageFile {
	return dapp.packageFile
}

func newDapp() *Dapp {
	dapp := &Dapp{}
	return dapp
}

type DappManager struct {
	mutex       *sync.Mutex
	keys        map[string]string
	dapps       map[string]dapps.Dapp
	rm          scripting.RuntimeManager
	server      network.Server
	runningDapp dapps.Dapp
	mm          modules.ModuleManager
	fio         files.FileIO
	//	hashDB *leveldb.DB
}

func NewDappManager(dc decerver.Decerver) dapps.DappManager {
	dm := &DappManager{}
	dm.keys = make(map[string]string)
	dm.dapps = make(map[string]dapps.Dapp)
	dm.mutex = &sync.Mutex{}
	dm.rm = dc.RuntimeManager()
	dm.mm = dc.ModuleManager()
	dm.server = dc.Server()
	dm.fio = dc.FileIO()
	return dm
}

func (dm *DappManager) RegisterDapps(directory, dbDir string) error {
	//	dbDir = path.Join(dbDir,"dapp_stored_hashes")
	//	dc.hashDB, _ = leveldb.OpenFile(dbDir,nil)
	//	defer dc.hashDB.Close()
	logger.Println("Registering dapps")
	files, err := ioutil.ReadDir(directory)

	if err != nil {
		return err
	}

	if len(files) == 0 {
		logger.Println("No dapps can be found.")
		return nil
	}

	for _, fileInfo := range files {
		if fileInfo.IsDir() {
			pth := path.Join(directory, fileInfo.Name())
			dm.RegisterDapp(pth)
		}
	}
	logger.Println("Done registering dapps.")
	return nil
}

func (dm *DappManager) RegisterDapp(dir string) {

	pkDir := path.Join(dir, dapps.PACKAGE_FILE_NAME)
	_, errPfn := os.Stat(pkDir)
	if errPfn != nil {
		logger.Printf("Error loading 'package.json' for dapp '%s'. Skipping...\n", dir)
		logger.Println(errPfn.Error())
		return
	}

	pkBts, errP := ioutil.ReadFile(pkDir)

	if errP != nil {
		logger.Printf("Error loading 'package.json' for dapp '%s'. Skipping...\n", dir)
		logger.Println(errP.Error())
		return
	}

	packageFile := &dapps.PackageFile{}
	pkUnmErr := json.Unmarshal(pkBts, packageFile)

	if pkUnmErr != nil {
		logger.Printf("The 'package.json' file for dapp '%s' is corrupted. Skipping...\n", dir)
		logger.Println(pkUnmErr.Error())
		return
	}

	idxDir := path.Join(dir, dapps.INDEX_FILE_NAME)
	_, errIf := os.Stat(idxDir)

	if errIf != nil {
		logger.Printf("Cannot find an 'index.html' file for dapp '%s'. Skipping...\n", dir)
		logger.Println(errIf.Error())
		return
	}

	modelDir := path.Join(dir, dapps.MODELS_FOLDER_NAME)

	modelFi, errMfi := os.Stat(modelDir)
	logger.Print("## Registering dapp: " + packageFile.Name + " ##")
	if errMfi != nil {
		logger.Printf("Error loading 'models' directory for dapp '%s'. Skipping.\n", dir)
		logger.Println(errMfi.Error())
		return
	}

	if !modelFi.IsDir() {
		logger.Printf("Error loading 'models' directory for dapp '%s': Not a directory.\n", dir)
		return
	}

	files, err := ioutil.ReadDir(modelDir)
	if err != nil {
		logger.Printf("Error loading 'models' directory for dapp '%s': Not a directory.\n", dir)
		return
	}

	if len(files) == 0 {
		logger.Printf("No models in model dir for app '%s', skipping.\n", dir)
		return
	}

	// Look for a config.json where loding order is defined. If there
	// is no such file, we just load them alphabetically.

	loConf := path.Join(modelDir, dapps.LOADING_ORDER_FILE_NAME)
	_, errLoc := os.Stat(loConf)
	if errLoc != nil {
		logger.Printf("Error loading 'config.json' for dapp '%s' models js loading. Skipping...\n", dir)
		logger.Println(errLoc.Error())
		return
	}

	locBts, errL := ioutil.ReadFile(loConf)

	if errL != nil {
		logger.Printf("Error loading 'config.json' for dapp '%s' models js loading. Skipping...\n", dir)
		logger.Println(errL.Error())
		return
	}

	loadConf := &dapps.LoadOrderConfig{}
	lcUnmErr := json.Unmarshal(locBts, loadConf)

	if lcUnmErr != nil {
		logger.Printf("The 'config.json' file for dapp '%s' model loading is corrupted. Skipping...\n", dir)
		logger.Println(pkUnmErr.Error())
		return
	}

	if len(loadConf.LoadingOrder) == 0 {
		logger.Printf("The loading order file list in the 'config.json' file for dapp '%s' model loading contains no files. Skipping...\n", dir)
		return
	}

	models := make([]string, 0)

	// TODO recursively and perhaps also a require.js type load file
	// to ensure the proper loading order.
	for _, mfName := range loadConf.LoadingOrder {
		fp := path.Join(modelDir, mfName)
		/* if fileInfo.IsDir() {
			logger.Println("Action models cannot be loaded recursively (yet). Skipping directory: " + fp)
			// Skip for now.
			continue
		}
		*/

		if strings.ToLower(path.Ext(fp)) != ".js" {
			//fmt.Println("[Dapp Registry] Skipping non .js file: " + fp)
			continue
		}

		fileBts, errFile := ioutil.ReadFile(fp)
		if errFile != nil {
			logger.Println("Error reading javascript file: " + fp)
		}

		jsFile := string(fileBts)

		// Catch some parse errors early on.
		_, errParse := parser.ParseFile(nil, "", jsFile, 0)

		if errParse != nil {
			logger.Printf("Error parsing javascript file '%s'. DUMP: %s\nError: %s\n", mfName, jsFile, errParse.Error())
			logger.Println("Skipping dapp: " + dir)
			return
		}

		logger.Printf("Loaded javascript file '%s'\n", path.Base(fp))

		models = append(models, jsFile)

	}

	// Create the dapp object and set it up.
	dapp := newDapp()
	dapp.path = dir
	dapp.packageFile = packageFile
	dapp.models = models

	/*

		TODO: Dapp verification has been postponed. Leaving this in, but


		// Hash the dapp files and check.
		hash := dc.HashApp(modelDir)

		if hash == nil {
			logger.Println("Failed to get hash of dapp files, skipping. Dapp: " + dir)
			return
		}

		oldHash, errH := dc.hashDB.Get([]byte(dapp.path), nil)

		if errH != nil {
			verify(dapp.path);
			logger.Printf("Adding new hash '%s' to folder '%s'.\n",hex.EncodeToString(hash),dir)
			dc.hashDB.Put([]byte(dapp.path),hash,nil)
		}

		if errH == nil && !bytes.Equal(hash,oldHash) {
			// TODO this is an old but updated dapp.
			logger.Printf("Hash mismatch: New: '%s', Old: '%s'.\n",hex.EncodeToString(hash),hex.EncodeToString(oldHash))
			verify(dapp.path);
			dc.hashDB.Put([]byte(dapp.path),hash,nil)
			dc.hashDB.Delete(oldHash,nil)
		} else {
			logger.Printf("Hash of '%s' matches the stored value: '%s'.\n", dir, hex.EncodeToString(hash))
		}
	*/

	dm.dapps[packageFile.Id] = dapp

	// Register the handlers right away.
	dm.server.RegisterDapp(dapp.packageFile.Id)

	return
}

// TODO check dependencies.
func (dm *DappManager) LoadDapp(dappId string) error {

	dm.mutex.Lock()
	defer dm.mutex.Unlock()
	dapp, ok := dm.dapps[dappId]
	if !ok {
		return errors.New("Error loading dapp: " + dappId + ". No dapp with that name has been registered.")
	}

	if dm.runningDapp != nil {
		if dm.runningDapp.PackageFile().Id == dappId {
			return errors.New("Error loading dapp - already running: " + dappId)
		}
		dm.UnloadDapp()
	}

	logger.Println("Loading dapp: " + dappId)

	rt := dm.rm.CreateRuntime(dappId)

	// Monk hack until we script
	deps := dapp.PackageFile().ModuleDependencies

	// TODO This module-implementation dependent block is only temporary. Once the
	// package.json file includes paths to config files, it will be possible to just
	// add the config file to the module when restarting it. No need to worry about
	// specific field names (or their types).
	if deps != nil {
		for _, d := range deps {
			if d.Name == "monk" {
				mData := d.Data
				if mData != nil {
					monkData := &dapps.MonkData{}
					err := json.Unmarshal(*mData, monkData)
					if err != nil {
						logger.Fatal("Blockchain will not work. Chain data for monk not available in dapp package file: " + dapp.PackageFile().Name)
					}
					monkMod, ok := dm.mm.Modules()["monk"]
					if !ok {
						logger.Fatal("Blockchain will not work. There is no Monk module.")
					}
					psAddr := monkData.PeerServerAddress
					addAndPort := strings.Split(psAddr, ":")
					if len(addAndPort) != 2 {
						logger.Fatal("Blockchain will not work. Malformed peerserver url: " + psAddr)
					}

					port, pErr := strconv.Atoi(addAndPort[1])
					if pErr != nil {
						logger.Fatal("Blockchain will not work. Malformed peerserver url (port not an integer)")
					}

					chainId := utils.StripHex(monkData.ChainId)
					monkMod.SetProperty("RootDir", chains.ComposeRoot("thelonious", chainId))
					monkMod.SetProperty("RemoteHost", addAndPort[0])
					monkMod.SetProperty("RemotePort", port)
					monkMod.SetProperty("ChainId", monkData.ChainId)
					
					monkMod.Restart()
					rc := monkData.RootContract
					if(len(rc) > 2){
						if(rc[1] != 'x'){
							rc = "0x" + rc;
						}
						logger.Println("Root contract: " + rc )
						rt.BindScriptObject("RootContract", rc)	
					}
				} else {
					logger.Fatal("Blockchain will not work. Chain data for monk not available in dapp package file: " + dapp.PackageFile().Name)
				}
			}
		}
	}

	for _, js := range dapp.Models() {
		rt.AddScript(js)
	}

	dm.runningDapp = dapp
	return nil
}

func (dm *DappManager) UnloadDapp() {
	// TODO cleanup
	dappId := dm.runningDapp.PackageFile().Id
	logger.Println("Unregistering dapp: " + dappId)
	dm.rm.RemoveRuntime(dappId)
	dm.runningDapp = nil
}

/*
func (dc *DappRegistry) HashApp(dir string) []byte {
	logger.Println("Hashing models folder: " + dir)
	hashes := dc.HashDir(dir)
	if hashes == nil {
		return nil
	}
	hash := sha1.Sum(hashes)
	return hash[:]
}
*/

/*
func (dc *DappRegistry) HashDir(directory string) []byte {
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		logger.Println(err.Error())
		return nil
	}
	if len(files) == 0 {
		logger.Println("No files in directory: " + directory)
		return nil
	}
	hashes := make([]byte, 0)
	for _, fileInfo := range files {
		if fileInfo.IsDir() {
			// This is a dapp
			pth := path.Join(directory, fileInfo.Name())
			hs := dc.HashDir(pth)
			if hs != nil {
				hashes = append(hashes, hs...)
			} else {
				return nil
			}
		} else {
			fBts, errF := ioutil.ReadFile(path.Join(directory, fileInfo.Name()))
			if errF != nil {
				logger.Printf(" Error loading '%s', skipping...\n", fileInfo.Name())
				logger.Println(errF.Error())
				return nil
			}
			hash := sha1.Sum(fBts)
			hashes = append(hashes, hash[:]...)
		}
	}
	return hashes
}
*/

func (dm *DappManager) DappList() []*dapps.DappInfo {
	arr := make([]*dapps.DappInfo, len(dm.dapps))
	ctr := 0
	for _, dapp := range dm.dapps {
		arr[ctr] = dapps.DappInfoFromPackageFile(dapp.PackageFile())
		ctr++
	}
	return arr
}

/*
func getVerification(string dappName) bool {

}
*/
