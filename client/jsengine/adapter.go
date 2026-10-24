package jsengine

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	//"reflect"
	//"sort"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
)

var vmpool = sync.Pool{
	New: func() interface{} {
		vm := goja.New()
		registry := new(require.Registry)
		registry.Enable(vm)
		console.Enable(vm)

		return vm
	},
}

/*/ 创建动态结构体，添加namespace参数处理嵌套结构的命名冲突
func createDynamicStruct(data map[string]interface{}, namespace ...string) (interface{}, error) {
	var fields []reflect.StructField

	// 获取排序后的键以保证结构体字段顺序一致
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	// 对键进行排序，确保生成结构体字段顺序一致
	sort.Strings(keys)

	for _, name := range keys { // 使用排序后的键
		value := data[name]
		fieldType, err := getFieldType(value, append(namespace, name)...)
		if err != nil {
			return nil, err
		}

		fieldName := getUniqueFieldName(name, namespace...)
		field := reflect.StructField{
			Name: fieldName,
			Type: fieldType,
			Tag:  reflect.StructTag(fmt.Sprintf(`json:"%s"`, name)),
		}
		fields = append(fields, field)
	}

	structType := reflect.StructOf(fields)
	structValue := reflect.New(structType).Elem()

	// 同样使用排序后的键来设置字段值
	for _, name := range keys {
		value := data[name]
		fieldName := getUniqueFieldName(name, namespace...)
		field, exists := structType.FieldByName(fieldName)
		if !exists {
			return nil, fmt.Errorf("字段 %s 不存在", fieldName)
		}
		fieldIndex := field.Index[0]
		if err := setFieldValue(structValue.Field(fieldIndex), value, append(namespace, name)...); err != nil {
			return nil, err
		}
	}

	return structValue.Interface(), nil
}

// 生成唯一字段名
func getUniqueFieldName(name string, namespace ...string) string {
	if len(namespace) == 0 {
		return toCamelCase(name)
	}
	namespaceStr := strings.Join(namespace[:len(namespace)-1], "_")
	if namespaceStr == "" {
		return toCamelCase(name)
	}
	return toCamelCase(namespaceStr) + "_" + toCamelCase(name)
}

// 确定字段类型，改进数组处理和 nil 值处理
func getFieldType(value interface{}, namespace ...string) (reflect.Type, error) {
	// 检查值是否为 nil，如果是，返回 interface{} 类型以避免 panic
	if value == nil {
		return reflect.TypeOf((*interface{})(nil)).Elem(), nil
	}

	switch v := value.(type) {
	case string:
		return reflect.TypeOf(""), nil
	case int, int64:
		return reflect.TypeOf(int64(0)), nil
	case float64, float32:
		return reflect.TypeOf(float64(0)), nil
	case bool:
		return reflect.TypeOf(bool(false)), nil
	case map[string]interface{}:
		nestedStruct, err := createDynamicStruct(v, namespace...)
		if err != nil {
			return nil, err
		}
		return reflect.TypeOf(nestedStruct), nil
	case []interface{}:
		return getSliceType(v, namespace...)
	default:
		return nil, fmt.Errorf("不支持的类型: %T", v)
	}
}

// 专门处理数组类型，确定统一的元素类型
func getSliceType(slice []interface{}, namespace ...string) (reflect.Type, error) {
	if len(slice) == 0 {
		// 空数组默认使用interface{}类型
		return reflect.TypeOf([]interface{}{}), nil
	}

	// 确定数组中所有元素的公共类型
	commonType, err := getCommonElementType(slice)
	if err != nil {
		return nil, err
	}

	// 如果是嵌套结构，递归处理 - 使用"elem"作为统一命名空间
	if nestedType, ok := commonType.(map[string]interface{}); ok {
		nestedStruct, err := createDynamicStruct(nestedType, append(namespace, "elem")...)
		if err != nil {
			return nil, err
		}
		return reflect.SliceOf(reflect.TypeOf(nestedStruct)), nil
	}

	// 处理基本类型数组
	return reflect.SliceOf(reflect.TypeOf(commonType)), nil
}

// 找到数组中所有元素的公共类型
func getCommonElementType(slice []interface{}) (interface{}, error) {
	if len(slice) == 0 {
		return nil, fmt.Errorf("空数组无法确定元素类型")
	}

	// 取第一个元素的类型作为基准
	baseType := reflect.TypeOf(slice[0])
	commonValue := slice[0]

	// 检查所有元素是否与基准类型兼容
	for _, elem := range slice[1:] {
		elemType := reflect.TypeOf(elem)

		// 如果类型不同，尝试找到更通用的类型
		if elemType != baseType {
			// 数字类型之间的兼容处理
			if isNumberType(baseType) && isNumberType(elemType) {
				// 统一为float64处理数字类型混合的情况
				baseType = reflect.TypeOf(float64(0))
				commonValue = float64(0)
			} else {
				// 不同类型且非数字，统一为interface{}
				return []interface{}{}[0], nil
			}
		}
	}

	return commonValue, nil
}

// 判断是否为数字类型
func isNumberType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

// 设置字段值，改进数组处理并添加错误返回，添加对nil的处理
func setFieldValue(field reflect.Value, value interface{}, namespace ...string) error {
	if !field.CanSet() {
		return fmt.Errorf("无法设置字段值")
	}

	// 处理nil值情况
	if value == nil {
		// 如果字段类型是interface{}，可以直接设置为nil
		if field.Kind() == reflect.Interface {
			field.Set(reflect.Zero(field.Type()))
			return nil
		}
		// 对于其他类型，尝试设置其零值
		field.Set(reflect.Zero(field.Type()))
		return nil
	}

	switch v := value.(type) {
	case string:
		field.SetString(v)
	case int:
		field.SetInt(int64(v))
	case int64:
		field.SetInt(v)
	case float64:
		if field.Kind() == reflect.Int64 {
			field.SetInt(int64(v))
		} else {
			field.SetFloat(v)
		}
	case float32:
		field.SetFloat(float64(v))
	case bool:
		field.SetBool(v)
	case map[string]interface{}:
		nestedStruct, err := createDynamicStruct(v, namespace...)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(nestedStruct))
	case []interface{}:
		return setSliceValue(field, v, namespace...)
	default:
		return fmt.Errorf("不支持的字段值类型: %T", v)
	}
	return nil
}

// 修改setBasicTypeValue函数，支持nil值
func setBasicTypeValue(field reflect.Value, value interface{}) error {
	// 处理nil值
	if value == nil {
		field.Set(reflect.Zero(field.Type()))
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		if str, ok := value.(string); ok {
			field.SetString(str)
			return nil
		}
	case reflect.Int64:
		switch v := value.(type) {
		case int:
			field.SetInt(int64(v))
			return nil
		case int64:
			field.SetInt(v)
			return nil
		case float64:
			field.SetInt(int64(v))
			return nil
		}
	case reflect.Float64:
		switch v := value.(type) {
		case float64:
			field.SetFloat(v)
			return nil
		case int:
			field.SetFloat(float64(v))
			return nil
		case int64:
			field.SetFloat(float64(v))
			return nil
		}
	case reflect.Bool:
		if b, ok := value.(bool); ok {
			field.SetBool(b)
			return nil
		}
	case reflect.Interface:
		field.Set(reflect.ValueOf(value))
		return nil
	}
	return fmt.Errorf("无法将 %T 类型的值设置到 %s 类型的字段", value, field.Kind())
}

// 专门处理数组值的设置
func setSliceValue(field reflect.Value, slice []interface{}, namespace ...string) error {
	sliceType := field.Type()
	if sliceType.Kind() != reflect.Slice {
		return fmt.Errorf("字段不是数组类型")
	}

	elemType := sliceType.Elem()
	newSlice := reflect.MakeSlice(sliceType, len(slice), len(slice))

	for i, elem := range slice {
		elemValue := reflect.New(elemType).Elem()

		// 根据元素类型进行相应转换
		if elemType.Kind() == reflect.Struct {
			// 嵌套结构体处理 - 使用"elem"作为统一命名空间，而不是"index%d"
			nestedMap, ok := elem.(map[string]interface{})
			if !ok {
				return fmt.Errorf("数组元素不是map类型，无法转换为结构体")
			}
			nestedStruct, err := createDynamicStruct(nestedMap, append(namespace, "elem")...)
			if err != nil {
				return err
			}
			elemValue.Set(reflect.ValueOf(nestedStruct))
		} else {
			// 基本类型处理
			if err := setBasicTypeValue(elemValue, elem); err != nil {
				return err
			}
		}

		newSlice.Index(i).Set(elemValue)
	}

	field.Set(newSlice)
	return nil
}

// 转换为驼峰命名
func toCamelCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}*/

type Program struct {
	program *goja.Program
}

// 加载模型adpater,执行预编译
func LoadLlmAdpater(lmname string) (*Program, error) {
	bytes, err := os.ReadFile("./" + lmname + ".adapter")
	if err != nil {
		return nil, err
	}
	jsCode := strings.TrimSpace(string(bytes))
	proc, err := goja.Compile("adapter.js", "(function() {"+jsCode+"\n})()", true)
	if err != nil {
		return nil, err
	}
	return &Program{program: proc}, nil
}

func (pro *Program) ConvertRequest(header map[string]string, openai []byte) (out []byte, err error) {
	// 1. 创建 JS 虚拟机
	vm := vmpool.Get().(*goja.Runtime)
	// 将 VM 放回池中以供将来重用
	defer vmpool.Put(vm)

	//requestMap := map[string]interface{}{}

	// 解析现有请求体到map
	var openaiMap map[string]interface{}
	if err := json.Unmarshal(openai, &openaiMap); err != nil {
		return nil, fmt.Errorf("error unmarshaling request body: %w", err)
	}
	vm.Set("request", openaiMap)
	vm.Set("response", nil)
	vm.Set("header", header)
	//vm.Set("openai", requestMap)
	result, err := vm.RunProgram(pro.program)
	if err != nil {
		fmt.Printf("执行JavaScript失败: %v\n", err)
		return nil, err
	}

	// 检查返回结果类型是否为map[string]interface{}
	resultMap, ok := result.Export().(map[string]interface{})
	if !ok {
		fmt.Printf("JavaScript返回非预期类型: %T\n", result)
		return nil, err
	}
	// Map to json[]byte
	body, err := json.Marshal(resultMap)
	return body, err
}

func (pro *Program) ConvertResponse(response []byte, stream bool) (out []byte, err error) {
	// 1. 创建 JS 虚拟机
	vm := vmpool.Get().(*goja.Runtime)
	// 将 VM 放回池中以供将来重用
	defer vmpool.Put(vm)

	//openaiMap := map[string]interface{}{}

	// 解析现有响应体到map
	var responseMap map[string]interface{}
	if err := json.Unmarshal(response, &responseMap); err != nil {
		return nil, fmt.Errorf("error unmarshaling request body: %w", err)
	}
	responseMap["stream"] = stream
	vm.Set("request", nil)
	vm.Set("response", responseMap)
	//vm.Set("openai", openaiMap)
	result, err := vm.RunProgram(pro.program)
	if err != nil {
		fmt.Printf("执行JavaScript失败: %v\n", err)
		return nil, err
	}
	// 检查返回结果类型是否为map[string]interface{}
	resultMap, ok := result.Export().(map[string]interface{})
	if !ok {
		fmt.Printf("JavaScript返回非预期类型: %T\n", result)
		return nil, err
	}
	// Map to json[]byte
	body, err := json.Marshal(resultMap)
	return body, err
}
