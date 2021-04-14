package render

import (
	"errors"
	"fmt"
	"github.com/flosch/pongo2/v4"
	"github.com/leekchan/accounting"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

func init() {
	pongo2.RegisterFilter("price", filterPrice)
	pongo2.RegisterFilter("sort", filterSort)
	pongo2.ReplaceFilter("date", filterDate)
	pongo2.RegisterFilter("keys", filterKeys)
	pongo2.RegisterFilter("map", filterMap)
}

func filterPrice(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in filterPrice", r)
		}
	}()
	if !in.IsFloat() {
		return nil, &pongo2.Error{
			Sender:    "filter:price",
			OrigError: errors.New("filter input argument must be of type 'float64'"),
		}
	}
	ac := accounting.Accounting{Symbol: "&euro;", Precision: 2, Thousand: ".", Decimal: ","}

	if param.IsString() {
		paramString := param.String()
		params := strings.Split(paramString, ",")

		if len(params) > 0 {
			ac.Symbol = params[0]
		}
		if len(params) > 1 {

			if i, err := strconv.Atoi(params[1]); err != nil {
				return nil, &pongo2.Error{
					Sender:    "filter:price",
					OrigError: fmt.Errorf("filter precision input should be an numberic value %s given", params[1]),
				}
			} else {
				ac.Precision = i
			}
		}

		if len(params) > 2 {
			ac.Thousand = params[0]
		}

		if len(params) > 3 {
			ac.Decimal = params[0]
		}
	}

	return pongo2.AsSafeValue(ac.FormatMoney(in.Float() / 100)), nil
}

func filterDate(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in filterPrice", r)
		}
	}()
	var t time.Time

	if in.IsString() {
		z := in.String()
		for _, format := range []string{"2006-01-02 15:04", "2006-01-02 15:04:05", "2006-01-02 15:04:05.000000", time.RFC3339} {
			if parsedTime, err := time.Parse(format, z); err == nil {
				t = parsedTime
				break
			}
		}
	} else if timeValue, isTime := in.Interface().(time.Time); isTime {
		t = timeValue
	} else {
		return nil, &pongo2.Error{
			Sender:    "filter:date",
			OrigError: errors.New("filter input argument must be of type 'time.Time'"),
		}
	}

	requestedLayout := param.String()

	// build a golang date string
	table := map[string]string{
		"d": "02",
		"D": "Mon",
		"j": "2",
		"l": "Monday",
		"N": "", // TODO: ISO-8601 numeric representation of the day of the week (added in PHP 5.1.0)
		"S": "", // TODO: English ordinal suffix for the day of the month, 2 characters
		"w": "", // TODO: Numeric representation of the day of the week
		"z": "", // TODO: The day of the year (starting from 0)
		"W": "", // TODO: ISO-8601 week number of year, weeks starting on Monday (added in PHP 4.1.0)
		"F": "January",
		"m": "01",
		"M": "Jan",
		"n": "1",
		"t": "", // TODO: Number of days in the given month
		"L": "", // TODO: Whether it's a leap year
		"o": "", // TODO: ISO-8601 year number. This has the same value as Y, except that if the ISO week number (W) belongs to the previous or next year, that year is used instead. (added in PHP 5.1.0)
		"Y": "2006",
		"y": "06",
		"a": "pm",
		"A": "PM",
		"B": "", // TODO: Swatch Internet time (is this even still a thing?!)
		"g": "3",
		"G": "15",
		"h": "03",
		"H": "15",
		"i": "04",
		"s": "05",
		"u": "000000",
		"e": "", // TODO: Timezone identifier (added in PHP 5.1.0)
		"I": "", // TODO: Whether or not the date is in daylight saving time
		"O": "-0700",
		"P": "-07:00",
		"T": "MST",
		"c": "2006-01-02T15:04:05-07:00",
		"r": "Mon, 02 Jan 2006 15:04:05 -0700",
		"U": "", // TODO: Seconds since the Unix Epoch (January 1 1970 00:00:00 GMT)
	}
	var layout string

	maxLen := len(requestedLayout)
	for i := 0; i < maxLen; i++ {
		char := string(requestedLayout[i])
		if t, ok := table[char]; ok {
			layout += t
			continue
		}
		if "\\" == char && i < maxLen-1 {
			layout += string(requestedLayout[i+1])
			continue
		}
		layout += char
	}

	return pongo2.AsValue(t.Format(layout)), nil
}

func filterSort(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in filterPrice", r)
		}
	}()
	if !in.CanSlice() {
		return nil, &pongo2.Error{
			Sender:    "filter:sort",
			OrigError: errors.New("cannot sort the input"),
		}
	}

	var nodes []string
	var isDescending bool
	if param.IsNil() {

	} else if !param.IsString() {
		return nil, &pongo2.Error{
			Sender:    "filter:sort",
			OrigError: errors.New("param shoud be a string"),
		}
	} else {

		runes := []rune(param.String())

		if runes[0] == '-' {
			isDescending = true
			runes = runes[1:]
		}
		nodes = strings.Split(string(runes), ".")
	}

	method := func(i, j int) bool {
		var valueA interface{}
		var valueB interface{}

		if len(nodes) > 0 {
			valueI, err := walk(reflect.ValueOf(in.Index(i).Interface()), nodes)

			if err != nil {
				return false
			}
			valueJ, err := walk(reflect.ValueOf(in.Index(j).Interface()), nodes)
			if err != nil {
				return true
			}
			valueA = valueI
			valueB = valueJ
		} else {
			valueA = in.Index(i).Interface()
			valueB = in.Index(j).Interface()

		}

		AStringValue, AisStringValue := valueA.(string)
		BStringValue, BisStringValue := valueB.(string)

		if AisStringValue && BisStringValue {
			if isDescending {
				return AStringValue > BStringValue
			}
			return AStringValue < BStringValue
		}

		AFloatValue, AisFloatValue := valueA.(float64)
		BFloatValue, BisFloatValue := valueB.(float64)

		if AisFloatValue && BisFloatValue {
			if isDescending {
				return AFloatValue > BFloatValue
			}
			return AFloatValue < BFloatValue
		}

		return false
	}

	slice := in.Interface()
	sort.Slice(slice, method)

	if isDescending {

	}

	return pongo2.AsValue(slice), nil
}

func filterKeys(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {

	z := in.Interface()
	_ = z
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in filterPrice", r)
		}
	}()
	if in.CanSlice() {
		var keys []interface{}
		keyFn := func(idx, count int, key, value *pongo2.Value) bool {
			keys = append(keys, key.Interface())
			return true
		}

		in.Iterate(keyFn, func() {})

		return pongo2.AsValue(keys), nil
	} else if reflect.ValueOf(in.Interface()).Kind() == reflect.Map {
		var keys []interface{}
		for _, value := range reflect.ValueOf(in.Interface()).MapKeys() {
			keys = append(keys, value.String())

		}
		return pongo2.AsValue(keys), nil
	} else {
		return nil, &pongo2.Error{
			Sender:    "filter:keys",
			OrigError: errors.New("cannot return keys the input"),
		}
	}

}

func filterMap(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in filterPrice", r)
		}
	}()
	if !param.IsString() {
		return nil, &pongo2.Error{
			Sender:    "filter:map",
			OrigError: errors.New("param shoud be a string"),
		}
	}

	options := strings.Split(param.String(), ",")

	if len(options) != 2 {
		return nil, &pongo2.Error{
			Sender:    "filter:map",
			OrigError: errors.New("param  should contain the option `keyNodes`,`value` in a dotted notation"),
		}
	}

	keyNodes := strings.Split(options[0], ".")
	valueNodes := strings.Split(options[1], ".")

	keys, _ := walk2(reflect.ValueOf(in.Interface()), keyNodes)
	values, _ := walk2(reflect.ValueOf(in.Interface()), valueNodes)

	data := make(map[string]interface{}, in.Len())

	for i := 0; i < len(keys.([]interface{})); i++ {
		data[keys.([]interface{})[i].(string)] = values.([]interface{})[i]
	}

	return pongo2.AsValue(data), nil

}

func walk(value reflect.Value, nodes []string) (interface{}, error) {

	key, nextNodes := nodes[0], nodes[1:]

	kind := value.Kind()
	switch kind {
	case reflect.Map:
		for _, v := range value.MapKeys() {
			if v.String() == key {
				if len(nextNodes) == 0 {
					return value.MapIndex(v).Elem().Interface(), nil
				}
				return walk(value.MapIndex(v).Elem(), nextNodes)
			}
		}
		return value.Interface(), nil
	case reflect.Slice, reflect.Array:
		if i, err := strconv.Atoi(key); err != nil {
			return nil, err
		} else {
			if len(nextNodes) == 0 {
				return value.Index(i).Elem().Interface(), nil
			}
			return walk(value.Index(i).Elem(), nextNodes)
		}
	default:
		val := value.Interface()
		return val, nil
	}

}

func walk2(value reflect.Value, nodes []string) (interface{}, error) {

	kind := value.Kind()
	switch kind {
	case reflect.Map:
		key, nextNodes := nodes[0], nodes[1:]
		for _, v := range value.MapKeys() {
			if v.String() == key {
				if len(nextNodes) == 0 {
					return value.MapIndex(v).Elem().Interface(), nil
				}
				return walk2(value.MapIndex(v).Elem(), nextNodes)
			}
		}
		return value.Interface(), nil
	case reflect.Slice, reflect.Array:
		ref := make([]interface{}, value.Len())
		for i := 0; i < value.Len(); i++ {
			var v interface{}
			if len(nodes) == 0 {
				indexValue, err := walk2(value.Index(i).Elem(), nodes)
				if err != nil {
					return nil, err
				}
				v = indexValue

			} else {
				indexValue, err := walk2(value.Index(i).Elem(), nodes)
				if err != nil {
					return nil, err
				}
				v = indexValue
			}
			ref[i] = v

		}
		return ref, nil
	default:
		val := value.Interface()
		return val, nil
	}

}
