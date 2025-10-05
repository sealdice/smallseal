package exts

import (
	"encoding/binary"
	"hash/fnv"
	"os"
	"runtime"
	"time"
	"unsafe"

	ds "github.com/sealdice/dicescript"
	"golang.org/x/exp/rand"

	"github.com/sealdice/smallseal/dice/types"
)

type RollExtraFlags struct {
	BigFailDiceOn      bool
	DisableLoadVarname bool  // 不允许加载变量，这是为了防止遇到 .r XXX 被当做属性读取，而不是“由于XXX，骰出了”
	CocVarNumberMode   bool  // 特殊的变量模式，此时这种类型的变量“力量50”被读取为50，而解析的文本被算作“力量”，如果没有后面的数字则正常进行
	CocDefaultAttrOn   bool  // 启用COC的默认属性值，如攀爬20等
	IgnoreDiv0         bool  // 当div0时暂不报错
	DisableValueBuff   bool  // 不计算buff值
	DNDAttrReadMod     bool  // 基础属性被读取为调整值，仅在检定时使用
	DNDAttrReadDC      bool  // 将力量豁免读取为力量再计算豁免值
	DefaultDiceSideNum int64 // 默认骰子面数
	DisableBitwiseOp   bool  // 禁用位运算，用于st
	DisableBlock       bool  // 禁用语句块(实际还禁用赋值语句，暂定名如此)
	DisableNumDice     bool  // 禁用r2d语法
	DisableBPDice      bool  // 禁用bp语法
	DisableCrossDice   bool  // 禁用双十字骰
	DisableDicePool    bool  // 禁用骰池

	V2Only     bool                                                                       // 仅使用v2
	V1Only     bool                                                                       // 仅使用v1，如果v1 v2两个选项同开，会走v1
	vmDepth    int64                                                                      // 层数
	StCallback func(_type string, name string, val *ds.VMValue, op string, detail string) // st回调
}

type VMResultV2m struct {
	*ds.VMValue
	vm        *ds.Context
	cocPrefix string
	errV2     error
}

func (r *VMResultV2m) GetAsmText() string {
	return r.vm.GetAsmText()
}

func (r *VMResultV2m) IsCalculated() bool {
	return r.vm.IsCalculateExists() || r.vm.IsComputedLoaded
}

func (r *VMResultV2m) GetRestInput() string {
	return r.vm.RestInput
}

func (r *VMResultV2m) GetMatched() string {
	return r.vm.Matched
}

func (r *VMResultV2m) GetCocPrefix() string {
	return r.cocPrefix
}

func (r *VMResultV2m) ToString() string {
	return r.VMValue.ToString()
}

// DiceExprEvalBase 向下兼容执行，首先尝试使用V2执行表达式
//
// Deprecated: 不建议用，纯兼容旧版
// 注: 考虑快速构建移植兼容性，先保留这玩意
func DiceExprEvalBase(ctx *types.MsgContext, s string, flags RollExtraFlags) (*VMResultV2m, string, error) {
	vm := ctx.GetVM()
	vm.Ret = nil
	vm.Error = nil

	vm.Config.DisableStmts = flags.DisableBlock
	vm.Config.IgnoreDiv0 = flags.IgnoreDiv0
	vm.Config.DiceMaxMode = flags.BigFailDiceOn

	var cocFlagVarPrefix string
	if flags.CocVarNumberMode {
		setCocPrefixReadForVM(ctx, func(val string) {
			cocFlagVarPrefix = val
		})
	}

	s = CompatibleReplace(ctx, s)

	err := vm.Run(s)
	if err != nil {
		return nil, "", err
	}
	return &VMResultV2m{
		VMValue:   vm.Ret,
		vm:        vm,
		cocPrefix: cocFlagVarPrefix,
		errV2:     err,
	}, vm.GetDetailText(), nil
}

// generateRandSeed 生成一个随机种子，由当前时间戳、对象指针、进程ID和堆栈信息组成
func generateRandSeed() uint64 {
	timestamp := time.Now().UnixNano()

	type tempObj struct{ val int }
	obj := tempObj{val: 42}
	objPtr := uint64(uintptr(unsafe.Pointer(&obj)))

	pid := uint64(os.Getpid())

	buf := make([]byte, 1024)
	n := runtime.Stack(buf, true)
	stackInfo := buf[:n]

	h := fnv.New64a()

	_ = binary.Write(h, binary.LittleEndian, timestamp)
	_ = binary.Write(h, binary.LittleEndian, objPtr)
	_ = binary.Write(h, binary.LittleEndian, pid)
	_, _ = h.Write(stackInfo)

	return h.Sum64()
}

var randSource = rand.NewSource(generateRandSeed()).(*rand.PCGSource)

func DiceRoll64x(src *rand.PCGSource, dicePoints int64) int64 { //nolint:revive
	if src == nil {
		src = randSource
	}
	val := ds.Roll(src, ds.IntType(dicePoints), 0)
	return int64(val)
}

func DiceRoll64(dicePoints int64) int64 { //nolint:revive
	return DiceRoll64x(nil, dicePoints)
}
