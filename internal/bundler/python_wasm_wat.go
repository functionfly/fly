package bundler

import (
	"fmt"
	"strings"
)

// generatePythonFunctions includes the necessary Python execution functions as WAT code
func (e *PythonCodeEmbedder) generatePythonFunctions() string {
	return `
  ;; Initialize Python code (called during module setup)
  (func $init_python_code (param $code_ptr i32) (param $code_len i32)
    ;; Store reference to embedded Python code and length
    local.get $code_ptr
    global.set $embedded_python_code_ptr
    local.get $code_len
    global.set $embedded_python_code_len

    ;; Mark code as available
    i32.const 1
    global.set $python_code_available
  )

  ;; Execute Python handler with extracted parameters
  (func $execute_python_handler (param $parameters i32) (result i32)
    (local $python_code_ptr i32)
    (local $handler_func i32)
    (local $execution_result i32)

    ;; 1. Load the embedded Python code
    call $load_embedded_python_code
    local.set $python_code_ptr

    ;; Check if Python code was loaded successfully
    local.get $python_code_ptr
    i32.eqz
    if
      ;; Return -1 if no Python code available
      i32.const -1
      return
    end

    ;; 2. Find the handler function in the Python code
    local.get $python_code_ptr
    call $find_handler_function
    local.set $handler_func

    ;; Check if handler function was found
    local.get $handler_func
    i32.eqz
    if
      ;; Return -2 if handler not found
      i32.const -2
      return
    end

    ;; 3. Execute the handler function with the parameters
    local.get $handler_func
    local.get $parameters
    call $execute_handler_with_event
    local.set $execution_result

    ;; 4. Return the execution result
    local.get $execution_result
  )

  ;; Execute from stdin - for _start function entry point
  (func $execute_from_stdin (result i32)
    (local $input_ptr i32)
    (local $input_len i32)

    ;; Read input from WASI stdin (stub - would need proper WASI import)
    ;; For now, use empty input
    i32.const 0
    local.set $input_ptr
    i32.const 0
    local.set $input_len

    ;; Call execute with the input and return the result
    local.get $input_ptr
    local.get $input_len
    call $execute_python_handler
    ;; Return the result from execute_python_handler
    return
  )

  ;; Load embedded Python code from WASM data section
  (func $load_embedded_python_code (result i32)
    (local $code_ptr i32)
    (local $code_len i32)
    (local $result_ptr i32)
    (local $scan_ptr i32)

    ;; Check if we have initialized Python code via globals first
    global.get $python_code_available
    if
      ;; Build [ptr, len] struct so find_handler_function can read it
      i32.const 8
      call $alloc
      local.set $result_ptr
      global.get $embedded_python_code_ptr
      local.get $result_ptr
      i32.store
      global.get $embedded_python_code_len
      local.get $result_ptr
      i32.const 4
      i32.add
      i32.store
      local.get $result_ptr
      return
    end

    ;; Try to access Python code from embedded data section
    call $find_python_data_section
    local.set $code_ptr

    ;; Check if we found a valid data section
    local.get $code_ptr
    i32.const 0
    i32.ne
    if
    else
      ;; Fallback: try scanning known data section locations
      call $scan_known_data_locations
      local.set $code_ptr

      local.get $code_ptr
      i32.const 0
      i32.ne
      if
      else
        ;; No embedded Python code found
        i32.const 0
        return
      end
    end

    ;; At this point, $code_ptr points to the start of Python code
    ;; Create a string structure for the embedded code
    i32.const 8
    call $alloc
    local.set $result_ptr

    ;; Calculate length by finding null terminator (with safety bound: one page = 64KB)
    local.get $code_ptr
    local.set $scan_ptr
    block $out
      loop $find_null
        local.get $scan_ptr
        i32.const 0x10000
        i32.ge_u
        br_if $out
        local.get $scan_ptr
        i32.load8_u
        i32.eqz
        br_if $out
        local.get $scan_ptr
        i32.const 1
        i32.add
        local.set $scan_ptr
        br $find_null
      end
    end

    ;; Calculate length (scan_ptr - code_ptr)
    local.get $scan_ptr
    local.get $code_ptr
    i32.sub
    local.set $code_len

    ;; Store pointer to embedded code
    local.get $result_ptr
    local.get $code_ptr
    i32.store

    ;; Store code length
    local.get $result_ptr
    i32.const 4
    i32.add
    local.get $code_len
    i32.store

    local.get $result_ptr
    return
  )

  ;; Find Python code in data section table
  (func $find_python_data_section (result i32)
    (local $table_ptr i32)
    (local $entry_count i32)
    (local $entry_offset i32)
    (local $entry_type i32)

    ;; Get data section table location
    global.get $data_section_table_start
    local.set $table_ptr

    ;; Verify table magic number
    local.get $table_ptr
    i32.load
    i32.const 0x03020100
    i32.ne
    if
      i32.const 0
      return
    end

    ;; Get entry count
    local.get $table_ptr
    i32.const 4
    i32.add
    i32.load
    local.set $entry_count

    ;; Scan through table entries
    i32.const 0
    local.set $entry_offset

    (loop $scan_entries
      local.get $entry_offset
      local.get $entry_count
      i32.lt_u
      if
        ;; Calculate entry offset (8 + entry_offset * 16); each entry is 16 bytes
        i32.const 8
        local.get $entry_offset
        i32.const 16
        i32.mul
        i32.add
        local.set $table_ptr

        ;; Get entry type (offset + 8)
        local.get $table_ptr
        i32.const 8
        i32.add
        i32.load
        local.set $entry_type

        ;; Check if this is a Python code section (type 1)
        local.get $entry_type
        i32.const 1
        i32.eq
        if
          ;; Found Python code section, get its offset
          local.get $table_ptr
          i32.load
          return
        end

        ;; Move to next entry
        local.get $entry_offset
        i32.const 1
        i32.add
        local.set $entry_offset
        br $scan_entries
      end
    )

    ;; No Python code section found in table
    i32.const 0
  )

  ;; Fallback: scan known data section locations
  (func $scan_known_data_locations (result i32)
    ;; Try main Python data section
    global.get $data_section_python_main
    call $check_data_section_for_python
    if
      global.get $data_section_python_main
      return
    end

    ;; No Python code found
    i32.const 0
  )

  ;; Check if a specific data section contains Python code
  (func $check_data_section_for_python (param $offset i32) (result i32)
    local.get $offset
    i32.const 0
    i32.lt_s
    if
      i32.const 0
      return
    end

    ;; Check if this offset contains Python code
    local.get $offset
    call $is_python_code_start
  )

  ;; Helper: Check if memory location contains Python code start
  (func $is_python_code_start (param $ptr i32) (result i32)
    (local $char1 i32)
    (local $char2 i32)
    (local $char3 i32)

    ;; Get first few characters
    local.get $ptr
    i32.load8_u
    local.set $char1

    local.get $ptr
    i32.const 1
    i32.add
    i32.load8_u
    local.set $char2

    local.get $ptr
    i32.const 2
    i32.add
    i32.load8_u
    local.set $char3

    ;; Check for "def" (function definition)
    local.get $char1
    i32.const 100
    i32.eq
    local.get $char2
    i32.const 101
    i32.eq
    i32.and
    local.get $char3
    i32.const 102
    i32.eq
    i32.and
    if
      i32.const 1
      return
    end

    ;; Check for "imp" (import)
    local.get $char1
    i32.const 105
    i32.eq
    local.get $char2
    i32.const 109
    i32.eq
    i32.and
    local.get $char3
    i32.const 112
    i32.eq
    i32.and
    if
      i32.const 1
      return
    end

    ;; Check for "#" (comment)
    local.get $char1
    i32.const 35
    i32.eq
    if
      i32.const 1
      return
    end

    ;; Not Python code
    i32.const 0
  )

  ;; Find handler function in Python code
  (func $find_handler_function (param $python_code_ptr i32) (result i32)
    (local $actual_code_ptr i32)
    (local $code_len i32)

    ;; Check if code pointer is valid
    local.get $python_code_ptr
    i32.eqz
    if
      i32.const 0
      return
    end

    ;; Extract actual code pointer and length
    local.get $python_code_ptr
    i32.load
    local.set $actual_code_ptr

    local.get $python_code_ptr
    i32.const 4
    i32.add
    i32.load
    local.set $code_len

    ;; Verify we have valid code
    local.get $actual_code_ptr
    i32.eqz
    if
      i32.const 0
      return
    end

    ;; Call the parser to find the handler function
    local.get $actual_code_ptr
    local.get $code_len
    call $parse_handler_function
    return
  )

  ;; Parse Python code to find handler function
  (func $parse_handler_function (param $code_ptr i32) (param $code_len i32) (result i32)
    (local $scan i32)
    (local $end_ptr i32)
    ;; Simple pattern matching for "def handler(" - scan from code_ptr
    local.get $code_ptr
    local.set $scan
    ;; End of valid range: code_ptr + code_len
    local.get $code_ptr
    local.get $code_len
    i32.add
    local.set $end_ptr

    (loop $scan_code
      ;; Need at least 5 bytes remaining (def h) to avoid out-of-bounds load
      local.get $end_ptr
      local.get $scan
      i32.sub
      i32.const 5
      i32.ge_u
      if
        ;; Check for 'd' (start of "def ")
        local.get $scan
        i32.load8_u
        i32.const 100  ;; 'd'
        i32.eq
        if
          ;; Check if followed by "ef handler("
          local.get $scan
          i32.const 1
          i32.add
          i32.load8_u
          i32.const 101  ;; 'e'
          i32.eq
          if
            local.get $scan
            i32.const 2
            i32.add
            i32.load8_u
            i32.const 102  ;; 'f'
            i32.eq
            if
              ;; Check for space after "def"
              local.get $scan
              i32.const 3
              i32.add
              i32.load8_u
              i32.const 32  ;; space
              i32.eq
              if
                ;; Check for "handler"
                local.get $scan
                i32.const 4
                i32.add
                i32.load8_u
                i32.const 104  ;; 'h'
                i32.eq
                if
                  ;; Found "def handler", return position
                  local.get $scan
                  return
                end
              end
            end
          end
        end
      end

      local.get $scan
      i32.const 1
      i32.add
      local.set $scan
      local.get $scan
      local.get $end_ptr
      i32.lt_u
      br_if $scan_code
    )

    ;; Handler not found
    i32.const 0
  )

  ;; Execute handler function with event parameters
  (func $execute_handler_with_event (param $handler_func i32) (param $event_params i32) (result i32)
    (local $result_ptr i32)
    (local $python_code_ptr i32)
    (local $function_body_ptr i32)
    (local $execution_result i32)

    ;; Get the Python code that contains this handler
    call $load_embedded_python_code
    local.set $python_code_ptr

    ;; Check if we have Python code
    local.get $python_code_ptr
    i32.eqz
    if
      call $create_error_response
      return
    end

    ;; Extract the handler function body from the Python code
    local.get $python_code_ptr
    local.get $handler_func
    call $extract_handler_function_body
    local.set $function_body_ptr

    ;; Check if we successfully extracted the function body
    local.get $function_body_ptr
    i32.eqz
    if
      ;; Return -3 if body extraction failed
      i32.const -3
      return
    end

    ;; Execute the Python function body with event parameters
    local.get $function_body_ptr
    local.get $event_params
    call $execute_python_function_body
    local.set $execution_result

    ;; Create result structure with actual execution result
    call $create_execution_result
    local.set $result_ptr

    ;; Store execution status (success)
    local.get $result_ptr
    i32.const 1
    i32.store

    ;; Store reference to input parameters
    local.get $result_ptr
    i32.const 4
    i32.add
    local.get $event_params
    i32.store

    ;; Store actual execution result
    local.get $result_ptr
    i32.const 8
    i32.add
    local.get $execution_result
    i32.store

    local.get $result_ptr
  )

  ;; Extract handler function body from Python code
  ;; Receives code [ptr, len] struct and handler position; bounds all reads to code range
  (func $extract_handler_function_body (param $code_struct i32) (param $handler_pos i32) (result i32)
    (local $code_ptr i32)
    (local $code_len i32)
    (local $code_end i32)
    (local $max_scan i32)
    (local $body_start i32)
    (local $i i32)
    (local $char i32)
    (local $body_len i32)
    (local $result_ptr i32)

    ;; Load code ptr and length from [ptr, len] struct
    local.get $code_struct
    i32.load
    local.set $code_ptr
    local.get $code_struct
    i32.const 4
    i32.add
    i32.load
    local.set $code_len
    local.get $code_ptr
    local.get $code_len
    i32.add
    local.set $code_end
    ;; max_scan = min(code_end, handler_pos + 4096)
    local.get $handler_pos
    i32.const 4096
    i32.add
    local.set $max_scan
    local.get $max_scan
    local.get $code_end
    i32.gt_u
    if
      local.get $code_end
      local.set $max_scan
    end

    ;; Find the start of function body (after the colon)
    local.get $handler_pos
    local.set $i

    ;; Skip to the colon (bounded by max_scan to avoid out-of-bounds read)
    block $out
      loop $find_colon
        local.get $i
        local.get $max_scan
        i32.ge_u
        if
          i32.const 0
          return
        end

        local.get $i
        i32.load8_u
        local.set $char

        ;; Look for ':' character - exit loop when found
        local.get $char
        i32.const 58
        i32.eq
        if
          local.get $i
          i32.const 1
          i32.add
          local.set $body_start
          br $out
        end

        local.get $i
        i32.const 1
        i32.add
        local.set $i
        br $find_colon
      end
    end

    ;; Body length = min(100, code_end - body_start) to avoid reading past code
    local.get $code_end
    local.get $body_start
    i32.sub
    local.set $body_len
    local.get $body_len
    i32.const 100
    i32.gt_u
    if
      i32.const 100
      local.set $body_len
    end

    ;; Create result structure [body_ptr, body_len]
    i32.const 8
    call $alloc
    local.set $result_ptr

    local.get $result_ptr
    local.get $body_start
    i32.store

    local.get $result_ptr
    i32.const 4
    i32.add
    local.get $body_len
    i32.store

    local.get $result_ptr
  )

  ;; Execute Python function body with event parameters
  (func $execute_python_function_body (param $body_ptr i32) (param $event_params i32) (result i32)
    (local $result_ptr i32)
    (local $i i32)
    (local $char i32)

    ;; Extract actual body pointer from string structure [ptr, len]
    local.get $body_ptr
    i32.load
    local.set $result_ptr

    ;; Get body length
    local.get $body_ptr
    i32.const 4
    i32.add
    i32.load
    local.set $body_ptr

    ;; Scan through function body looking for "return" statements
    local.get $result_ptr
    local.set $i

    (loop $scan_body
      ;; Bounds: i < result_ptr + body_ptr
      local.get $i
      local.get $result_ptr
      local.get $body_ptr
      i32.add
      i32.lt_u
      if
        local.get $i
        i32.load8_u
        local.set $char

        ;; Look for 'r' (start of "return") - need 6 bytes remaining to read "return"
        local.get $result_ptr
        local.get $body_ptr
        i32.add
        local.get $i
        i32.sub
        i32.const 6
        i32.ge_u
        if
          local.get $char
          i32.const 114  ;; 'r'
          i32.eq
          if
            ;; Check if this is "return"
            local.get $i
            i32.const 1
            i32.add
            i32.load8_u
            i32.const 101  ;; 'e'
            i32.eq
            if
              local.get $i
              i32.const 2
              i32.add
              i32.load8_u
              i32.const 116  ;; 't'
              i32.eq
              if
                local.get $i
                i32.const 3
                i32.add
                i32.load8_u
                i32.const 117  ;; 'u'
                i32.eq
                if
                  local.get $i
                  i32.const 4
                  i32.add
                  i32.load8_u
                  i32.const 114  ;; 'r'
                  i32.eq
                  if
                  local.get $i
                  i32.const 5
                  i32.add
                  i32.load8_u
                  i32.const 110  ;; 'n'
                  i32.eq
                  if
                    ;; Found "return", now extract the return value
                    local.get $i
                    i32.const 6  ;; Skip "return"
                    i32.add
                    i32.const 0  ;; No event params in this context
                    call $extract_return_value
                    return
                  end
                end
              end
            end
          end
        end
        end

        local.get $i
        i32.const 1
        i32.add
        local.set $i
        br $scan_body
      end
    )

    ;; No return statement found, return default value (0)
    i32.const 0
  )

  ;; Extract return value from Python expression (simplified)
  (func $extract_return_value (param $expr_ptr i32) (param $event_params i32) (result i32)
    (local $i i32)
    (local $char i32)

    local.get $expr_ptr
    local.set $i

    ;; Skip whitespace (block so we can break out when non-whitespace found)
    block $out
      loop $skip_whitespace
        local.get $i
        i32.load8_u
        local.set $char

        local.get $char
        i32.const 32  ;; space
        i32.eq
        if
          local.get $i
          i32.const 1
          i32.add
          local.set $i
          br $skip_whitespace
        end

        local.get $char
        i32.const 10  ;; newline
        i32.eq
        if
          local.get $i
          i32.const 1
          i32.add
          local.set $i
          br $skip_whitespace
        end

        ;; Non-whitespace found, exit loop
        br $out
      end
    end

    ;; Check for simple cases
    local.get $char
    i32.const 34  ;; '"'
    i32.eq
    if
      ;; String literal - return pointer to it
      local.get $i
      return
    end

    local.get $char
    i32.const 123  ;; '{'
    i32.eq
    if
      ;; Object/dict - return pointer to it
      local.get $i
      return
    end

    local.get $char
    i32.const 91  ;; '['
    i32.eq
    if
      ;; Array/list - return pointer to it
      local.get $i
      return
    end

    ;; Try to parse numbers
    local.get $char
    i32.const 48  ;; '0'
    i32.ge_u
    local.get $char
    i32.const 57  ;; '9'
    i32.le_u
    i32.and
    if
      ;; Number literal - parse integer value
      local.get $i
      call $parse_integer
      return
    end

    ;; Check for boolean literals and None
    local.get $char
    i32.const 84  ;; 'T'
    i32.eq
    if
      ;; Check for "True"
      local.get $i
      i32.const 1
      i32.add
      i32.load8_u
      i32.const 114  ;; 'r'
      i32.eq
      if
        local.get $i
        i32.const 2
        i32.add
        i32.load8_u
        i32.const 117  ;; 'u'
        i32.eq
        if
          local.get $i
          i32.const 3
          i32.add
          i32.load8_u
          i32.const 101  ;; 'e'
          i32.eq
          if
            ;; Found "True", return 1
            i32.const 1
            return
          end
        end
      end
    end

    local.get $char
    i32.const 70  ;; 'F'
    i32.eq
    if
      ;; Check for "False"
      local.get $i
      i32.const 1
      i32.add
      i32.load8_u
      i32.const 97  ;; 'a'
      i32.eq
      if
        local.get $i
        i32.const 2
        i32.add
        i32.load8_u
        i32.const 108  ;; 'l'
        i32.eq
        if
          local.get $i
          i32.const 3
          i32.add
          i32.load8_u
          i32.const 115  ;; 's'
          i32.eq
          if
            local.get $i
            i32.const 4
            i32.add
            i32.load8_u
            i32.const 101  ;; 'e'
            i32.eq
            if
              ;; Found "False", return 0
              i32.const 0
              return
            end
          end
        end
      end
    end

    local.get $char
    i32.const 78  ;; 'N'
    i32.eq
    if
      ;; Check for "None"
      local.get $i
      i32.const 1
      i32.add
      i32.load8_u
      i32.const 111  ;; 'o'
      i32.eq
      if
        local.get $i
        i32.const 2
        i32.add
        i32.load8_u
        i32.const 110  ;; 'n'
        i32.eq
        if
          local.get $i
          i32.const 3
          i32.add
          i32.load8_u
          i32.const 101  ;; 'e'
          i32.eq
          if
            ;; Found "None", return -1 (special value for None)
            i32.const -1
            return
          end
        end
      end
    end

    ;; Check for variable references (event, context)
    local.get $char
    i32.const 101  ;; 'e'
    i32.eq
    if
      ;; Check for "event"
      local.get $i
      i32.const 1
      i32.add
      i32.load8_u
      i32.const 118  ;; 'v'
      i32.eq
      if
        local.get $i
        i32.const 2
        i32.add
        i32.load8_u
        i32.const 101  ;; 'e'
        i32.eq
        if
          local.get $i
          i32.const 3
          i32.add
          i32.load8_u
          i32.const 110  ;; 'n'
          i32.eq
          if
            local.get $i
            i32.const 4
            i32.add
            i32.load8_u
            i32.const 116  ;; 't'
            i32.eq
            if
              ;; Found "event" - return pointer to event parameters
              ;; This would be passed in as $event_params to the parent function
              ;; For now, return a placeholder pointer
              i32.const 1000  ;; Placeholder pointer to event data
              return
            end
          end
        end
      end
    end

    local.get $char
    i32.const 99  ;; 'c'
    i32.eq
    if
      ;; Check for "context"
      local.get $i
      i32.const 1
      i32.add
      i32.load8_u
      i32.const 111  ;; 'o'
      i32.eq
      if
        local.get $i
        i32.const 2
        i32.add
        i32.load8_u
        i32.const 110  ;; 'n'
        i32.eq
        if
          local.get $i
          i32.const 3
          i32.add
          i32.load8_u
          i32.const 116  ;; 't'
          i32.eq
          if
            local.get $i
            i32.const 4
            i32.add
            i32.load8_u
            i32.const 101  ;; 'e'
            i32.eq
            if
              local.get $i
              i32.const 5
              i32.add
              i32.load8_u
              i32.const 120  ;; 'x'
              i32.eq
              if
                local.get $i
                i32.const 6
                i32.add
                i32.load8_u
                i32.const 116  ;; 't'
                i32.eq
                if
                  ;; Found "context" - return pointer to context
                  i32.const 2000  ;; Placeholder pointer to context data
                  return
                end
              end
            end
          end
        end
      end
    end

    ;; For unhandled expressions, return 0
    i32.const 0
  )

  ;; Parse integer literal from string (simplified)
  (func $parse_integer (param $str_ptr i32) (result i32)
    (local $result i32)
    (local $char i32)
    (local $i i32)

    i32.const 0
    local.set $result
    local.get $str_ptr
    local.set $i

    ;; Parse digits until non-digit or end
    (loop $parse_loop
      local.get $i
      i32.load8_u
      local.set $char

      ;; Check if digit
      local.get $char
      i32.const 48  ;; '0'
      i32.ge_u
      local.get $char
      i32.const 57  ;; '9'
      i32.le_u
      i32.and
      if
        ;; Add digit to result: result = result * 10 + (char - '0')
        local.get $result
        i32.const 10
        i32.mul
        local.get $char
        i32.const 48
        i32.sub
        i32.add
        local.set $result

        local.get $i
        i32.const 1
        i32.add
        local.set $i
        br $parse_loop
      end
    )

    local.get $result
  )

  ;; Create execution result structure
  (func $create_execution_result (result i32)
    ;; Allocate result structure (status, input_ref, result_data)
    i32.const 12
    call $alloc
  )

  ;; Create error response
  (func $create_error_response (result i32)
    ;; Return a simple error code
    i32.const -1
  )`
}

// generateDataSectionTable creates the data section table WAT code
func (e *PythonCodeEmbedder) generateDataSectionTable() string {
	var tableParts []string

	// Magic number: \00\01\02\03 (0x03020100 in little endian)
	magic := `\00\01\02\03`
	count := len(e.dataSections)

	// Start building the table data
	tableData := magic + packInt32(count)

	// Add table entries (16 bytes each: offset, size, type, name_ptr)
	nameOffset := 256 // Start of name area after table
	for _, section := range e.dataSections {
		// Store section name
		nameData := fmt.Sprintf(`  (data (i32.const %d) "%s")`,
			nameOffset, escapeForWAT(section.Name+"\x00"))
		tableParts = append(tableParts, nameData)

		// Add table entry data
		tableData += packInt32(section.Offset)
		tableData += packInt32(section.Size)
		tableData += packInt32(section.Type)
		tableData += packInt32(nameOffset)

		nameOffset += len(section.Name) + 1
	}

	// Create the main table data section
	table := fmt.Sprintf(`  ;; Data section table - populated by build system
  ;; Format: [magic(4), count(4), entries...]
  ;; Each entry: [offset(4), size(4), type(4), name_ptr(4)]
  ;; Types: 1=python_code, 2=config, 3=strings, etc.
  (data (i32.const 0) "%s")`,
		tableData)

	// Add name data sections
	for _, nameSection := range tableParts {
		table += "\n" + nameSection
	}

	return table
}

// generateDataSections creates the actual data sections with user content
func (e *PythonCodeEmbedder) generateDataSections() string {
	var sections []string

	for _, section := range e.dataSections {
		// Escape the content for WAT format
		escapedContent := escapeForWAT(section.Content)

		sectionWAT := fmt.Sprintf(`  ;; %s data section
  (data (i32.const %d) "%s")`,
			section.Name,
			section.Offset,
			escapedContent)

		sections = append(sections, sectionWAT)
	}

	return strings.Join(sections, "\n")
}

// generateDataSectionOffsets creates global exports for data section offsets
func (e *PythonCodeEmbedder) generateDataSectionOffsets() string {
	var globals []string

	// Add standard globals
	globals = append(globals, "(global $data_section_table_start i32 (i32.const 0))")
	globals = append(globals, fmt.Sprintf("(global $data_section_table_size i32 (i32.const %d))", 8+len(e.dataSections)*16))

	// Add specific section offsets
	for _, section := range e.dataSections {
		if section.Type == 1 && section.Name == "python_main" {
			globals = append(globals, fmt.Sprintf("(global $data_section_python_main i32 (i32.const %d))", section.Offset))
		} else if section.Type == 2 && section.Name == "metadata" {
			globals = append(globals, fmt.Sprintf("(global $data_section_metadata i32 (i32.const %d))", section.Offset))
		}
	}

	return strings.Join(globals, "\n  ")
}
