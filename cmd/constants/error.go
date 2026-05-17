package constants

import "math"

const ERR_WRONG_NUMBER_OF_ARGS_PING = "ERR wrong number of arguments for 'ping' command"

const ERR_WRONG_NUMBER_OF_ARGS_GET = "ERR wrong number of arguments for 'get' command"
const ERR_WRONG_NUMBER_OF_ARGS_SET = "ERR wrong number of arguments for 'set' command"

const ERR_WRONG_NUMBER_OF_ARGS_LPUSH = "ERR wrong number of arguments for 'lpush' command"
const ERR_WRONG_NUMBER_OF_ARGS_RPUSH = "ERR wrong number of arguments for 'rpush' command"

const ERR_WRONG_NUMBER_OF_ARGS_LLEN = "ERR wrong number of arguments for 'llen' command"

const ERR_WRONG_NUMBER_OF_ARGS_LPOP = "ERR wrong number of arguments for 'lpop' command"
const ERR_WRONG_NUMBER_OF_ARGS_BLPOP = "ERR wrong number of arguments for 'blpop' command"

const ERR_WRONG_NUMBER_OF_ARGS_TYPE = "ERR wrong number of arguments for 'type' command"

const ERR_WRONG_NUMBER_OF_ARGS_XADD = "ERR wrong number of arguments for 'xadd' command"
const ERR_INVALID_ID_XADD_SMALLER = "ERR The ID specified in XADD is equal or smaller than the target stream top item"
const ERR_INVALID_ID_MUST_BE_GREATER_XADD = "ERR The ID specified in XADD must be greater than 0-0"
const ERR_WRONGTYPE_OPERATION = "WRONGTYPE Operation against a key holding the wrong kind of value"

const ERR_INVALID_STREAM_ID = "ERR Invalid stream ID specified as stream command argument"

const ERR_WRONG_NUMBER_OF_ARGS_XRANGE = "ERR wrong number of arguments for 'xrange' command"

const ERR_SYNTAX_ERROR = "ERR syntax error"

const MaxInt = math.MaxInt
