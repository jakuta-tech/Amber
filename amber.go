package main


import "gopkg.in/cheggaaa/pb.v1"
import "github.com/fatih/color"
import "strconv"
import "os/exec"
import "strings"
import "runtime"
import "fmt"
import "os"

const VERSION string = "1.0.0"


type PARAMETERS struct {
  inputName string 
  outputName string
  keySize int
  key []byte
  VP string 				// VirtualProtect address
  LLA string				// LoadLibraryA address
  GPA string				// GetProcAddress address
  verbose	bool
}

type PE struct {
  imageBase string
  subSystem string
  LLA string
  GPA string
  VP string
  mapView []byte
}

var Red *color.Color = color.New(color.FgRed)
var BoldRed *color.Color = Red.Add(color.Bold)
var	Blue *color.Color = color.New(color.FgBlue)
var	BoldBlue *color.Color = Blue.Add(color.Bold)
var	Yellow *color.Color = color.New(color.FgYellow)
var	BoldYellow *color.Color = Yellow.Add(color.Bold)
var	Green *color.Color = color.New(color.FgGreen)
var	BoldGreen *color.Color = Green.Add(color.Bold)


var progressBar *pb.ProgressBar
var parameters PARAMETERS
var pe PE

func main() {

  runtime.GOMAXPROCS(runtime.NumCPU())

  parameters.outputName = "crypted.exe"
  parameters.keySize = 8
  parameters.verbose = false

 	ARGS := os.Args[1:]
  if len(ARGS) == 0 || ARGS[0] == "--help" || ARGS[0] == "-h"{
    Banner()
    Help()
    os.Exit(0)
  }

  Banner()
  progressBar = pb.New(25)
  progressBar.SetWidth(80)
  progressBar.Start()

  if CheckRequirments() == false {
    BoldRed.Println("\n\n[!] ERROR: Amber is not installed properly (missing dependencies)")
    os.Exit(1)
  }


  progressBar.Increment()

  parameters.inputName = ARGS[0]

  for i := 0; i < len(ARGS); i++{
  	if ARGS[i] == "-ks" || ARGS[i] == "--keysize" {
  		ks, Err := strconv.Atoi(ARGS[i+1])
      if Err != nil {
        BoldRed.Println("[!] ERROR: Invalid key size.")
        fmt.Println(Err)
        os.Exit(1)
      }else{
        parameters.keySize = ks
      } 
  	}
  	if ARGS[i] == "-k" || ARGS[i] == "--key" {
  		parameters.key = []byte(ARGS[i+1]) 
  	}
  	if ARGS[i] == "-o" || ARGS[i] == "--out" {
  		parameters.outputName = ARGS[i+1]
  	}
    if ARGS[i] == "-v" || ARGS[i] == "--verbose" {
      parameters.verbose = true 
    }
  	if ARGS[i] == "--virtualprotect" {
  		parameters.VP = ARGS[i+1] 
  	}
  	if ARGS[i] == "--loadlibrarya" {
  		parameters.VP = ARGS[i+1] 
  	} 
  	if ARGS[i] == "--getprocaddress" {
  		parameters.VP = ARGS[i+1] 
  	}
  }

  if parameters.verbose == true {
    BoldYellow.Print("\n[*] File: ")
    BoldBlue.Println(parameters.inputName)
    BoldYellow.Print("\n[*] Output: ")
    BoldBlue.Println(parameters.outputName)
    BoldYellow.Print("[*] Verbose: ")
    BoldBlue.Println(parameters.verbose)
    BoldYellow.Print("[*] Key Size: ")
    BoldBlue.Println(parameters.keySize)
  }



  progressBar.Increment()

  InspectPE()
  BuildPayload()
  CryptPayload()
  CompileStub()
  CleanFiles()

  progressBar.FinishPrint("\n")

  BoldGreen.Println("[+] Operation successfull !")

}

func CompileStub() {

  if parameters.verbose == true {
    BoldYellow.Println("[*] Compiling Stub... ")
  }

  var CompileCommand string = ""
  if pe.subSystem == "(Windows GUI)"{
    CompileCommand = string("i686-w64-mingw32-g++-win32 Stub.cpp -Wl,--image-base=0x"+pe.imageBase+" -o "+parameters.outputName)  
  }else{
    CompileCommand = string("i686-w64-mingw32-g++-win32 Stub.cpp -Wl,--image-base=0x"+pe.imageBase+" -mwindows -o "+parameters.outputName)
  }
  mingw, Err := exec.Command("sh", "-c", CompileCommand).Output()
  if strings.Contains(string(mingw), "error") {
    BoldRed.Println("\n[!] ERROR: While compiling the stub :(")
    Red.Println(string(mingw))
    Red.Println(Err)
    CleanFiles()
    os.Exit(1)
  }

  if parameters.verbose == true {
    BoldYellow.Println(CompileCommand)
    BoldYellow.Println("\n[*] Striping pe file... ")
  }
  progressBar.Increment()
  exec.Command("sh", "-c", string("strip "+parameters.outputName)).Run()
  progressBar.Increment()
  exec.Command("sh", "-c", string("mkdir /tmp/Amber/ && mv "+parameters.outputName+" /tmp/Amber/")).Run()
  progressBar.Increment()
}



func InspectPE() {

  ls, Err := exec.Command("sh", "-c", string("ls  "+parameters.inputName)).Output()
  if (!strings.Contains(string(ls), parameters.inputName)) || (Err != nil)  {
    BoldRed.Println("\n[!] ERROR: Unable to locate file :(")
    Red.Println(string(ls))
    Red.Println(Err)
    os.Exit(1)
  }

  progressBar.Increment()

	magic, _ := exec.Command("sh", "-c", string("objdump -x "+parameters.inputName+"|grep Magic")).Output()
	if !strings.Contains(string(magic), "010b") {
		BoldRed.Println("\n[!] ERROR: File is not a valid PE")
		BoldRed.Println(string(magic))
		os.Exit(1)
	}
  progressBar.Increment()
	arch, _ := exec.Command("sh", "-c", string("objdump -x "+parameters.inputName+"|grep architecture")).Output()
	if !strings.Contains(string(arch), "i386"){
		BoldRed.Println("\n[!] ERROR: Unsupported file architecture (only 32 PE files supported)")
		BoldYellow.Println(string(arch))
		os.Exit(1)		
	}
  progressBar.Increment()
	imageBase, _ := exec.Command("sh", "-c", string("objdump -x "+parameters.inputName+"| grep ImageBase|tr -d \"ImageBase		\"|tr -d \"\\n\" ")).Output()
  pe.imageBase = string(imageBase)
  progressBar.Increment()
	subSystem, _ := exec.Command("sh", "-c", string("objdump -x "+parameters.inputName)).Output()
	if strings.Contains(string(subSystem), "(Windows GUI)") {
		pe.subSystem = "(Windows GUI)"
	}else{
		pe.subSystem = "(Windows CUI)"
	}
  progressBar.Increment()
	imports, _ := exec.Command("sh", "-c", string("objdump -x "+parameters.inputName+"|grep \"<none>\"")).Output()
	if len(imports) > 1 {
		BoldRed.Println("\n[!] ERROR: Incompatible PE file (file has unbounded import names)")
		BoldYellow.Println(string(imports))
		os.Exit(1)
	}
	progressBar.Increment()
	boundImports, _ := exec.Command("sh", "-c", string("objdump -x "+parameters.inputName+"|grep \"Bound Import Directory\" |tr -d \"Entry b \"|tr -d \"BoudImpoDieco\"")).Output()
	if string(boundImports) != "0000000000000000\n" {
		BoldRed.Println("\n[!] ERROR: Incompatible PE file (file has bounded imports)")
		BoldYellow.Println(string(boundImports))
		os.Exit(1)
	}
  progressBar.Increment()
  if parameters.verbose == true {
    BoldYellow.Print("\n[*] "+string(magic))
    BoldYellow.Print("[*] "+string(arch))
    BoldYellow.Println("[*] ImageBase: 0x"+pe.imageBase)
    BoldYellow.Println("[*] SubSystem: "+pe.subSystem)
  }
}

func BuildPayload() {

  MapPE, _ := exec.Command("sh", "-c", string("wine MapPE.exe "+parameters.inputName)).Output()
  progressBar.Increment()
  nasm := exec.Command("sh", "-c", "nasm -f bin ReplaceProcess.asm -o Payload")
  nasm.Stdout = os.Stdout
  nasm.Stderr = os.Stderr
  nasm.Run()
  progressBar.Increment()

  if parameters.verbose == true {
    _MapPE := strings.Split(string(MapPE), "github.com/egebalci/mappe")
    fmt.Println(string(_MapPE[1]))
  }

}


func CryptPayload() {
  if parameters.verbose == true {
    BoldYellow.Println("[*] Ciphering payload...")    
  }
  if len(parameters.key) != 0 {
    exec.Command("sh", "-c", string("./bitbender ^=\""+string(parameters.key)+"\" Payload")).Run()  
  }else{
    ks := strconv.Itoa(parameters.keySize)
    exec.Command("sh", "-c", "./bitbender ^ "+ks+" Payload").Run()
  }
  progressBar.Increment()  

  xxd := exec.Command("sh", "-c", "rm Payload && mv Payload.xor Payload && xxd -i Payload > PAYLOAD.h")
  xxd.Stdout = os.Stdout
  xxd.Stderr = os.Stderr
  xxd.Run()

  progressBar.Increment()  

  _xxd := exec.Command("sh", "-c", "xxd -i Payload.key > KEY.h")
  _xxd.Stdout = os.Stdout
  _xxd.Stderr = os.Stderr
  _xxd.Run()

  progressBar.Increment()  

  if parameters.verbose == true {
    key, _ := exec.Command("sh", "-c", "xxd -i Payload.key").Output() 
    BoldYellow.Println("[*] Payload ciphered with: ")
    BoldBlue.Println(string(key))    
  }
  
}

func CleanFiles() {

  exec.Command("sh", "-c", "rm Mem.dmp").Run()
  exec.Command("sh", "-c", "rm Payload").Run()
  exec.Command("sh", "-c", "rm Payload.xor").Run()
  exec.Command("sh", "-c", "rm Payload.key").Run()
}

func Help() {
	 var Help string = `Version : `+VERSION+`
Source : github.com/EgeBalci/Amber

USAGE: 
  amber file.exe [options]


OPTIONS:
  
  -k, --key [string]          Payload type to use (1-5)
  -ks, --keysize <length>        Size of the encryption key in bytes (Max:100/Min:4)
  -o, --out       <name>          Output file name
  -v, --verbose                   Verbose output mode
  -h, --help                      Show this massage

EXAMPLE:
  (Default settings if no option parameter passed)
  amber file.exe -ks 8 -o crypted.exe
`
  Green.Println(Help)

}

func CheckRequirments() (bool){

  CheckMingw, _ := exec.Command("sh", "-c", "i686-w64-mingw32-g++-win32 --version").Output()
  if (!strings.Contains(string(CheckMingw), "Copyright")) {
    return false
  }
  progressBar.Increment()
  CheckMingwDress, _ := exec.Command("sh", "-c", "i686-w64-mingw32-windres -V").Output()
  if (!strings.Contains(string(CheckMingwDress), "Copyright")) {
   	return false
  }
  progressBar.Increment()
 	CheckNasm, _ := exec.Command("sh", "-c", "nasm -h").Output()
  if (!strings.Contains(string(CheckNasm), "usage:")) {
    return false
  }
  progressBar.Increment()
  CheckStrip, _ := exec.Command("sh", "-c", "strip -V").Output()
  if (!strings.Contains(string(CheckStrip), "Copyright")) {
    return false
  }
  progressBar.Increment()
  CheckWine, _ := exec.Command("sh", "-c", "wine --help").Output()
  if (!strings.Contains(string(CheckWine), "Usage:")) {
    return false
  }
  progressBar.Increment()
  Checkbitbender, _ := exec.Command("sh", "-c", "./bitbender").Output()
  if (!strings.Contains(string(Checkbitbender), "USAGE:")) {
    return false
  }
  progressBar.Increment()
  CheckMapPE, _ := exec.Command("sh", "-c", "wine MapPE.exe").Output()
  if (!strings.Contains(string(CheckMapPE), "Usage:")) {
    return false
  }
  progressBar.Increment()
 	CheckXXD, _ := exec.Command("sh", "-c", "echo Ambe | xxd").Output()
  if (!strings.Contains(string(CheckXXD), "416d 6265 0a")) {
    return false
  }
  progressBar.Increment()
  CheckMultiLib, _ := exec.Command("sh", "-c", "apt-cache policy gcc-multilib").Output()
  if (strings.Contains(string(CheckMultiLib), "(none)")) {
    return false
  }
  progressBar.Increment()
	CheckMultiLibPlus, _ := exec.Command("sh", "-c", "apt-cache policy g++-multilib").Output()
  if (strings.Contains(string(CheckMultiLibPlus), "(none)")) {
    return false
  }
  progressBar.Increment()
	return true

}


func Banner() {
  var BANNER string = `

//   █████╗ ███╗   ███╗██████╗ ███████╗██████╗ 
//  ██╔══██╗████╗ ████║██╔══██╗██╔════╝██╔══██╗
//  ███████║██╔████╔██║██████╔╝█████╗  ██████╔╝
//  ██╔══██║██║╚██╔╝██║██╔══██╗██╔══╝  ██╔══██╗
//  ██║  ██║██║ ╚═╝ ██║██████╔╝███████╗██║  ██║
//  ╚═╝  ╚═╝╚═╝     ╚═╝╚═════╝ ╚══════╝╚═╝  ╚═╝
//  POC Crypter For ReplaceProcess                                             
    
# Version: `+VERSION+`
# Source: github.com/EgeBalci/Amber

  `
  BoldRed.Println(BANNER)
  
}
	