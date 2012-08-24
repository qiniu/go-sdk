package conf

var CLIENT_ID = "abcd0c7edcdf914228ed8aa7c6cee2f2bc6155e2"
var CLIENT_SECRET = "fc9ef8b171a74e197b17f85ba23799860ddf3b9c"

var REDIRECT_URI = "<RedirectURL>"
var AUTHORIZATION_ENDPOINT = "<AuthURL>"
var TOKEN_ENDPOINT = "https://acc.qbox.me/oauth2/token"

var QBOX_ACCESS_KEY = "4_odedBxmrAHiu4Y0Qp0HPG0NANCf6VAsAjWL_kO"
var QBOX_SECRET_KEY = "SrRuUVfDX6drVRvpyN8mv8Vcm9XnMZzlbDfvVfmE"

var FS_HOST = "https://fs.qbox.me"
var IO_HOST = "http://iovip.qbox.me"
var RS_HOST = "http://rs.qbox.me:10100"
var UP_HOST = "http://up.qbox.me"
//var UP_HOST = "http://0.0.0.0:11200"
var EU_HOST = "http://eu.qbox.me"
var PU_HOST = "http://pu.qbox.me:10200"
var UC_HOST = "http://uc.qbox.me"

var BLOCK_BITS uint = 22
var PUT_CHUNK_SIZE = 256 * 1024 // 256k
var PUT_RETRY_TIMES = 2
var RS_PUT = "/rs-put/"