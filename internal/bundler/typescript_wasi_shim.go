package bundler

import (
	"strings"
)

// WASIEnvironment represents the WASI environment for TypeScript WASM functions
type WASIEnvironment struct {
	// Environment variables
	Env map[string]string
	// Argument list
	Args []string
	// Working directory
	WorkingDir string
	// Preopen directories
	PreopenDirs []string
	// Stdin content (for testing)
	Stdin string
}

// NewWASIEnvironment creates a new WASI environment
func NewWASIEnvironment() *WASIEnvironment {
	return &WASIEnvironment{
		Env:         make(map[string]string),
		Args:        []string{},
		WorkingDir:  "/",
		PreopenDirs: []string{},
		Stdin:       "",
	}
}

// WASIFunction represents a WASI function signature
type WASIFunction struct {
	Name string
	// ParamTypes and ResultTypes use WASM value types: i32, i64, f32, f64
	ParamTypes  []string
	ResultTypes []string
}

// WASIShim generates a JavaScript WASI polyfill for TypeScript functions
type WASIShim struct {
	env *WASIEnvironment
}

// NewWASIShim creates a new WASI shim
func NewWASIShim() *WASIShim {
	return &WASIShim{
		env: NewWASIEnvironment(),
	}
}

// Generate returns the JavaScript code for the WASI polyfill
func (s *WASIShim) Generate() string {
	return `// FunctionFly WASI Polyfill for TypeScript WASM
(function() {
  'use strict';

  // WASI constants
  var WASI_ERRNO_SUCCESS = 0;
  var WASI_ERRNO_BADF = 8;
  var WASI_ERRNO_INVAL = 28;
  var WASI_ERRNO_NOMEM = 12;
  var WASI_STDIN_FILENO = 0;
  var WASI_STDOUT_FILENO = 1;
  var WASI_STDERR_FILENO = 2;

  // File descriptor table
  var fdTable = {
    0: { isatty: true, data: '' },
    1: { isatty: false, data: '' },
    2: { isatty: false, data: '' }
  };
  var nextFd = 3;

  // Memory for string storage
  var stringStorage = [];
  var stringStoragePtr = 0;

  // Convert JavaScript string to WASM pointer
  function stringToWasm(str, allocFunc) {
    var len = str.length;
    var ptr = allocFunc(len + 1);
    var mem = new Uint8Array(wasmMemory.buffer);

    for (var i = 0; i < len; i++) {
      mem[ptr + i] = str.charCodeAt(i) & 0xFF;
    }
    mem[ptr + len] = 0;

    return { ptr: ptr, len: len };
  }

  // Convert WASM pointer to JavaScript string
  function wasmToString(ptr, len) {
    if (ptr === 0 || len === 0) return '';
    var mem = new Uint8Array(wasmMemory.buffer);
    var result = '';
    var end = ptr + len;
    for (var i = ptr; i < end; i++) {
      if (mem[i] === 0) break;
      result += String.fromCharCode(mem[i]);
    }
    return result;
  }

  // fd_write - Write to a file descriptor
  // (fd: i32, iovs_ptr: i32, iovs_len: i32, nwritten_ptr: i32) -> i32
  function fd_write(fd, iovsPtr, iovsLen, nwrittenPtr) {
    if (fdTable[fd] === undefined) {
      return WASI_ERRNO_BADF;
    }

    var totalWritten = 0;
    var mem = new Uint8Array(wasmMemory.buffer);

    for (var i = 0; i < iovsLen; i++) {
      var iovPtr = iovsPtr + i * 8;
      var bufPtr = mem.readUint32LE(iovPtr);
      var bufLen = mem.readUint32LE(iovPtr + 4);

      var data = wasmToString(bufPtr, bufLen);
      fdTable[fd].data += data;
      totalWritten += bufLen;
    }

    // Write nwritten
    mem.writeUint32LE(nwrittenPtr, totalWritten);

    return WASI_ERRNO_SUCCESS;
  }

  // proc_exit - Terminate the process
  // (code: i32) -> void
  function proc_exit(code) {
    throw new Error('WASI proc_exit called with code: ' + code);
  }

  // environ_get - Get environment variables
  // (environ: i32, environBuf: i32) -> i32
  function environ_get(environPtr, environBufPtr) {
    var envVars = Object.keys(__wasiEnv);
    var mem = new Uint8Array(wasmMemory.buffer);
    var bufOffset = environBufPtr;

    for (var i = 0; i < envVars.length; i++) {
      var key = envVars[i];
      var value = __wasiEnv[key];
      var pair = key + '=' + value;
      var ptr = stringToWasm(pair, __wasiAlloc);

      // Store pointer in environ array
      mem.writeUint32LE(environPtr + i * 4, ptr.ptr);

      // Store string in buffer
      for (var j = 0; j < pair.length + 1; j++) {
        mem[bufOffset + j] = mem[ptr.ptr + j];
      }
      bufOffset += pair.length + 1;
    }

    return WASI_ERRNO_SUCCESS;
  }

  // environ_sizes_get - Get number and size of environment variables
  // (environCount: i32, environBufSize: i32) -> i32
  function environ_sizes_get(environCountPtr, environBufSizePtr) {
    var envVars = Object.keys(__wasiEnv);
    var totalSize = 0;

    for (var i = 0; i < envVars.length; i++) {
      var key = envVars[i];
      var value = __wasiEnv[key] || '';
      totalSize += key.length + value.length + 2; // +2 for '=' and null terminator
    }

    var mem = new Uint8Array(wasmMemory.buffer);
    mem.writeUint32LE(environCountPtr, envVars.length);
    mem.writeUint32LE(environBufSizePtr, totalSize);

    return WASI_ERRNO_SUCCESS;
  }

  // fd_seek - Seek file descriptor
  // (fd: i32, offset: i64, whence: i32, newOffsetPtr: i32) -> i32
  function fd_seek(fd, offset, whence, newOffsetPtr) {
    // Simplified: just return success
    return WASI_ERRNO_SUCCESS;
  }

  // fd_close - Close file descriptor
  // (fd: i32) -> i32
  function fd_close(fd) {
    if (fdTable[fd] === undefined) {
      return WASI_ERRNO_BADF;
    }
    delete fdTable[fd];
    return WASI_ERRNO_SUCCESS;
  }

  // fd_fdstat_get - Get file descriptor status
  // (fd: i32, statBuf: i32) -> i32
  function fd_fdstat_get(fd, statBufPtr) {
    if (fdTable[fd] === undefined) {
      return WASI_ERRNO_BADF;
    }

    // File type: character device (isatty) or regular file
    var fs_filetype = fdTable[fd].isatty ? 2 : 0; // _CHR = 2, _REG = 0

    var mem = new Uint8Array(wasmMemory.buffer);
    mem.writeUint8(statBufPtr, fs_filetype);  // fs_filetype
    mem.writeUint8(statBufPtr + 1, 0);          // fs_flags (unused)
    mem.writeUint16LE(statBufPtr + 2, 0);       // fs_rights_base (unused)
    mem.writeUint16LE(statBufPtr + 4, 0);       // fs_rights_inheriting (unused)

    return WASI_ERRNO_SUCCESS;
  }

  // path_open - Open a file
  // (dirfd: i32, pathPtr: i32, pathLen: i32, oflags: i32, fsRightsBase: i64,
  //  fsRightsInheriting: i64, fdFlags: i32, fdPtr: i32) -> i32
  function path_open(dirfd, pathPtr, pathLen, oflags, fsRightsBase, fsRightsInheriting, fdFlags, fdPtr) {
    var path = wasmToString(pathPtr, pathLen);

    // Only allow read of predefined paths
    if (path === '/dev/stdout' || path === '/dev/stderr') {
      var fd = path === '/dev/stdout' ? WASI_STDOUT_FILENO : WASI_STDERR_FILENO;
      var mem = new Uint8Array(wasmMemory.buffer);
      mem.writeUint32LE(fdPtr, fd);
      return WASI_ERRNO_SUCCESS;
    }

    return WASI_ERRNO_INVAL;
  }

  // Custom host functions for FunctionFly

  // kv_get - Get value from key-value store
  function kv_get(keyPtr, keyLen, valuePtrPtr) {
    var key = wasmToString(keyPtr, keyLen);
    var value = __kvStoreGet(key);

    if (value === null || value === undefined) {
      return WASI_ERRNO_INVAL; // Key not found
    }

    var result = stringToWasm(value, __wasiAlloc);
    var mem = new Uint8Array(wasmMemory.buffer);
    mem.writeUint32LE(valuePtrPtr, result.ptr);
    mem.writeUint32LE(valuePtrPtr + 4, result.len);

    return WASI_ERRNO_SUCCESS;
  }

  // kv_set - Set value in key-value store
  function kv_set(keyPtr, keyLen, valuePtr, valueLen) {
    var key = wasmToString(keyPtr, keyLen);
    var value = wasmToString(valuePtr, valueLen);

    __kvStoreSet(key, value);

    return WASI_ERRNO_SUCCESS;
  }

  // fetch - Make HTTP request
  function fetch(urlPtr, urlLen, reqPtr, reqLen, respPtrPtr, respLenPtr) {
    var url = wasmToString(urlPtr, urlLen);
    var reqData = wasmToString(reqPtr, reqLen);

    var request = JSON.parse(reqData);
    request.url = url;

    // Make fetch request through host
    var response = __functionfly_fetch(request.url, request);
    var responseData = JSON.stringify(response);

    var result = stringToWasm(responseData, __wasiAlloc);
    var mem = new Uint8Array(wasmMemory.buffer);
    mem.writeUint32LE(respPtrPtr, result.ptr);
    mem.writeUint32LE(respLenPtr, result.len);

    return WASI_ERRNO_SUCCESS;
  }

  // get_secret - Get secret from vault
  function get_secret(keyPtr, keyLen, valuePtrPtr, valueLenPtr) {
    var key = wasmToString(keyPtr, keyLen);
    var value = __getSecret(key);

    if (value === null || value === undefined) {
      return WASI_ERRNO_INVAL;
    }

    var result = stringToWasm(value, __wasiAlloc);
    var mem = new Uint8Array(wasmMemory.buffer);
    mem.writeUint32LE(valuePtrPtr, result.ptr);
    mem.writeUint32LE(valueLenPtr, result.len);

    return WASI_ERRNO_SUCCESS;
  }

  // Export WASI functions
  var wasi = {
    fd_write: fd_write,
    proc_exit: proc_exit,
    environ_get: environ_get,
    environ_sizes_get: environ_sizes_get,
    fd_seek: fd_seek,
    fd_close: fd_close,
    fd_fdstat_get: fd_fdstat_get,
    path_open: path_open,
    // Custom functions
    kv_get: kv_get,
    kv_set: kv_set,
    fetch: fetch,
    get_secret: get_secret
  };

  // Store environment
  var __wasiEnv = {};

  // Initialize environment
  function initWasiEnv(env) {
    __wasiEnv = env || {};
  }

  // Export
  if (typeof module !== 'undefined' && module.exports) {
    module.exports = { wasi: wasi, initWasiEnv: initWasiEnv };
  }
  if (typeof window !== 'undefined') {
    window.FunctionFlyWASI = { wasi: wasi, initWasiEnv: initWasiEnv };
  }
  if (typeof self !== 'undefined') {
    self.FunctionFlyWASI = { wasi: wasi, initWasiEnv: initWasiEnv };
  }

  return { wasi: wasi, initWasiEnv: initWasiEnv };
})();
`
}

// GenerateWASIWrapper generates a complete WASI wrapper for TypeScript functions
func GenerateWASIWrapper(envVars map[string]string) string {
	// Initialize WASI shim
	shim := &WASIShim{
		env: NewWASIEnvironment(),
	}
	shim.env.Env = envVars
	return shim.Generate()
}

// WASIToTypeScript generates TypeScript type definitions for WASI
func WASIToTypeScript() string {
	return `// WASI TypeScript type definitions
// These types define the WASI interface available to TypeScript WASM functions

declare namespace WASI {
  interface Errno {
    SUCCESS: number;
    BADF: number;
    INVAL: number;
    NOMEM: number;
  }

  const errno: Errno;

  interface IOVec {
    buf: number;
    buf_len: number;
  }

  interface FdStat {
    fs_filetype: number;
    fs_flags: number;
    fs_rights_base: number;
    fs_rights_inheriting: number;
  }

  interface Environment {
    [key: string]: string;
  }

  // WASI functions
  function fd_write(fd: number, iovs: number, iovs_len: number, nwritten: number): number;
  function proc_exit(code: number): never;
  function environ_get(environ: number, environBuf: number): number;
  function environ_sizes_get(environCount: number, environBufSize: number): number;
  function fd_seek(fd: number, offset: bigint, whence: number, newOffset: number): number;
  function fd_close(fd: number): number;
  function fd_fdstat_get(fd: number, statBuf: number): number;
  function path_open(dirfd: number, path: number, pathLen: number, oflags: number,
                   fsRightsBase: bigint, fsRightsInheriting: bigint, fdFlags: number, fd: number): number;

  // Custom FunctionFly functions
  function kv_get(key: number, keyLen: number, valuePtr: number): number;
  function kv_set(key: number, keyLen: number, value: number, valueLen: number): number;
  function fetch(url: number, urlLen: number, req: number, reqLen: number,
                resp: number, respLen: number): number;
  function get_secret(key: number, keyLen: number, valuePtr: number, valueLen: number): number;
}

// Polyfill for when WASI is not available
declare var WASI: {
  new: (options?: { version?: string; env?: Record<string, string> }) => WASI;
};

export { WASI };
`
}

// GenerateTypeScriptWASI generates TypeScript type definitions for the WASI polyfill
func GenerateTypeScriptWASI() string {
	var buf strings.Builder

	buf.WriteString(`// FunctionFly WASI Polyfill TypeScript Definitions

export interface WASIEnv {
  [key: string]: string;
}

export interface WASIOptions {
  env?: WASIEnv;
  args?: string[];
  stdin?: string;
  workingDir?: string;
}

export interface IOVec {
  buf: number;
  buf_len: number;
}

export interface FdStat {
  fs_filetype: number;
  fs_flags: number;
  fs_rights_base: number;
  fs_rights_inheriting: number;
}

export const WASI_ERRNO_SUCCESS = 0;
export const WASI_ERRNO_BADF = 8;
export const WASI_ERRNO_INVAL = 28;
export const WASI_ERRNO_NOMEM = 12;

export const WASI_STDIN_FILENO = 0;
export const WASI_STDOUT_FILENO = 1;
export const WASI_STDERR_FILENO = 2;

// WASI core functions
export function fd_write(fd: number, iovs: number, iovs_len: number, nwritten: number): number;
export function proc_exit(code: number): never;
export function environ_get(environ: number, environBuf: number): number;
export function environ_sizes_get(environCount: number, environBufSize: number): number;
export function fd_seek(fd: number, offset: bigint, whence: number, newOffset: number): number;
export function fd_close(fd: number): number;
export function fd_fdstat_get(fd: number, statBuf: number): number;
export function path_open(dirfd: number, path: number, pathLen: number, oflags: number,
                         fsRightsBase: bigint, fsRightsInheriting: bigint, fdFlags: number, fd: number): number;

// Custom FunctionFly host functions
export function kv_get(keyPtr: number, keyLen: number, valuePtr: number): number;
export function kv_set(keyPtr: number, keyLen: number, valuePtr: number, valueLen: number): number;
export function fetch(urlPtr: number, urlLen: number, reqPtr: number, reqLen: number,
                     respPtr: number, respLen: number): number;
export function get_secret(keyPtr: number, keyLen: number, valuePtr: number, valueLenPtr: number): number;

export interface WASIModule {
  init(env?: WASIEnv, args?: string[]): void;
  wasi: {
    fd_write: typeof fd_write;
    proc_exit: typeof proc_exit;
    environ_get: typeof environ_get;
    environ_sizes_get: typeof environ_sizes_get;
    fd_seek: typeof fd_seek;
    fd_close: typeof fd_close;
    fd_fdstat_get: typeof fd_fdstat_get;
    path_open: typeof path_open;
    kv_get: typeof kv_get;
    kv_set: typeof kv_set;
    fetch: typeof fetch;
    get_secret: typeof get_secret;
  };
}

declare var FunctionFlyWASI: WASIModule;
export default FunctionFlyWASI;
`)

	return buf.String()
}

// GetWASIImports returns the necessary import statements for WASI
func GetWASIImports() string {
	return `// Import WASI polyfill
import { FunctionFlyWASI } from '@functionfly/wasi-polyfill';
`
}
