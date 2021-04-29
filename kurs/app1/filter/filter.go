package filter

import (
	"fmt"
	"regexp"
	"strings"
)

type FilterIf interface {
	Filter() (map[string]string, error)
}

type base string

const (
	OP_E   base = "="
	OP_M   base = ">"
	OP_L   base = "<"
	OP_ME  base = ">="
	OP_LE  base = "<="
	OP_OR  base = "OR"
	OP_AND base = "AND"
	OP_XOR base = "XOR"
)

// условие из кс
// column_name OP value [AND/OR column_name OP value] ...
// continent=’Asia’ AND date>’2020-04-14’
// todo make recursive condition setup and test (check)
// todo make no New filter for every row.(only setup filtercondition)
type Condition struct {
	colname    string
	oper       base
	value      string
	nextcond   *Condition
	nextcondop base
}

// filter object
type Filter struct {
	// query parsed
	Cond *Condition
	// map of cols mask
	ColsMask map[string]int
	// map of cols idx as per original order
	ColsIdx map[string]int
}

func (fl *Filter) New(args string, fileCols []string, flgCols []string) (*Filter, error) {
	return NewFilter(args, fileCols, flgCols)
}

// инициализация структуры - установление структуры - condition - условия отбора
func NewFilter(args string, fileCols, flgCols []string) (*Filter, error) {
	// todo init
	filter := new(Filter)

	colsMask, colsIdx := filter.ParseHeading(fileCols, flgCols)

	// cmd := strings.Split(args, " ")
	// special split doesnot split in doulble quotes
	r := regexp.MustCompile(`[^\s"']+|"[^"]*"`)
	cmd := r.FindAllString(args, -1)

	cond, err := filter.ParseCondition(cmd, colsMask) //use colsmask just to validate col_name in user query
	if err != nil {
		return nil, err
	}

	//setup Filter object
	return &Filter{
		ColsMask: colsMask,
		ColsIdx:  colsIdx,
		Cond:     cond,
		//wg:              new(sync.WaitGroup),
	}, nil
}

// parse flgCols and FileCols make maps for them
// 1)colsMask - map of cols which is user specified in flgCols
// 2)colsIdx - map of original index of columns as per it is in file (to restore original order in output)
func (fl *Filter) ParseHeading(fileCols, flgCols []string) (map[string]int, map[string]int) {

	colsMask := make(map[string]int)
	colsIdx := make(map[string]int)

	for i, col := range fileCols {
		//make key in map Mask
		colsMask[col] = 0
		colsIdx[col] = i
		// if no cols in flag mask all and continue
		if flgCols[0] == "" {
			colsMask[col] = 1
			continue
		}

		for _, flg := range flgCols {
			if col == flg {
				// if match make this key val 1
				colsMask[col] = 1
				break
			}
		}
	}
	return colsMask, colsIdx
}

func (fl *Filter) ParseCondition(cmd []string, colsMask map[string]int) (*Condition, error) {
	// parse cli to condition
	// "column_name OP value" Например, age > 40 AND status = “sick”
	cond := new(Condition)

	// first check column_name
	if _, ok := colsMask[cmd[0]]; ok {
		//column is ok
		cond.colname = cmd[0]
		//now check OP
		switch cmd[1] {
		case "=":
			cond.oper = OP_E
		case "<":
			cond.oper = OP_L
		case ">":
			cond.oper = OP_M
		case "<=":
			cond.oper = OP_LE
		case ">=":
			cond.oper = OP_ME
		default:
			err := fmt.Errorf("operand %s not found", cmd[1])
			return nil, err
		}
		// operand ok
		// now check value
		//if we have qoutes remove them
		cond.value = strings.Trim(cmd[2], "\"")

		// check if we have 4th member AND OR link b/w several conditions
		if len(cmd) > 4 {
			// check link b/w logical sets
			switch cmd[3] {
			case "AND":
				cond.nextcondop = OP_AND
			case "OR":
				cond.nextcondop = OP_OR
			case "XOR":
				cond.oper = OP_XOR
			default:
				err := fmt.Errorf("operand %s not found", cmd[3])
				return nil, err
			}
			//call it recursively till we get to end of all args each time we move by 4 positions in args - to next set
			var err error
			cond.nextcond, err = fl.ParseCondition(cmd[4:], colsMask)
			if err != nil {
				return nil, err
			}

		}

	} else {
		// create error
		err := fmt.Errorf("column %s not found", cmd[0])
		return nil, err
	}

	//all good return condition struct
	return cond, nil
}

// check one logical set like 'lla = 123' OR...
func (fl *Filter) Check(condition *Condition, cols map[string]string) bool {

	if val, ok := cols[condition.colname]; ok {
		val = strings.Trim(val, "\"")
		switch condition.oper {
		case OP_E:
			if val == condition.value {
				return true
			}
		case OP_M:
			if val > condition.value {
				return true
			}
		case OP_L:
			if val < condition.value {
				return true
			}
		case OP_ME:
			if val >= condition.value {
				return true
			}
		case OP_LE:
			if val <= condition.value {
				return true
			}
		default:
			return false
		}

	}
	// by default return false
	return false
}

// метод структуры - проверка и отбор данных берет мапу Cols и выдает такую же мапу выходных данных
func (fl *Filter) Filter(filerow []string) (string, error) {
	// process data which is
	// map of key - col name: col val in the row
	cols := make(map[string]string)

	for col, _ := range fl.ColsMask {
		//build up row to check against conditions
		cols[col] = filerow[fl.ColsIdx[col]]
	}
	//cols is map of 'col name: col val' in the one current row
	condition := fl.Cond // start
	//initial
	var res = false
	var prev_res = false
	var prev_nextcondop = ""
	// evaluate through all conditions linked in Cond struct
	for condition != nil {
		res = fl.Check(condition, cols)
		switch prev_nextcondop {
		case "AND":
			res = prev_res && res
		case "OR":
			res = prev_res || res
		case "XOR":
			res = prev_res != res
		}
		prev_res = res
		prev_nextcondop = string(condition.nextcondop)
		condition = condition.nextcond
	}

	if res {
		//condition is true print this cols formatted
		var outarr []string

		// return all row with matched one of the key:val
		// out := cols
		// format output as its in file comma separated val1,val2,... in this row

		for col, val := range cols {
			if fl.ColsMask[col] == 1 {
				outarr = stringsliceins(outarr, fl.ColsIdx[col], val)
			}
		}

		out := strings.Join(outarr, ",")
		return out, nil
	}

	err := fmt.Errorf("no data within condition found")
	return "", err
}

//its a shame go doesnt have this func yet as builtin
//insert element before pos in slice. if pos >= len(arr) insert into tail
func stringsliceins(arr []string, pos int, elem string) []string {
	if pos < 0 {
		pos = 0
	} else if pos >= len(arr) {
		pos = len(arr)
	}
	out := make([]string, len(arr)+1)
	copy(out[:pos], arr[:pos])
	out[pos] = elem
	copy(out[pos+1:], arr[pos:])
	return out
}
