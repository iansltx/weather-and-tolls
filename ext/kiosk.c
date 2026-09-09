/* PHP glue for the kiosk extension. This file is hand-maintained: it
 * propagates errors from the Go layer as PHP exceptions
 * (spl_ce_RuntimeException), mirroring the pattern used by FrankenPHP's own
 * mercure_publish/log functions. Do not regenerate with the extension
 * generator; the arginfo/stub files still describe the PHP API. */

#include <php.h>
#include <Zend/zend_API.h>
#include <Zend/zend_exceptions.h>
#include <Zend/zend_hash.h>
#include <Zend/zend_types.h>
#include <ext/spl/spl_exceptions.h>
#include <stdlib.h>

#include "kiosk.h"
#include "kiosk_arginfo.h"
#include "_cgo_export.h"

PHP_MINIT_FUNCTION(kiosk) {
    return SUCCESS;
}

zend_module_entry kiosk_module_entry = {STANDARD_MODULE_HEADER,
                                         "kiosk",
                                         ext_functions,             /* Functions */
                                         PHP_MINIT(kiosk),  /* MINIT */
                                         NULL,                      /* MSHUTDOWN */
                                         NULL,                      /* RINIT */
                                         NULL,                      /* RSHUTDOWN */
                                         NULL,                      /* MINFO */
                                         "1.0.0",                   /* Version */
                                         STANDARD_MODULE_PROPERTIES};
PHP_FUNCTION(kiosk_resolve_coords)
{
    zend_string *location = NULL;
    ZEND_PARSE_PARAMETERS_START(1, 1)
        Z_PARAM_STR(location)
    ZEND_PARSE_PARAMETERS_END();

    struct go_kiosk_resolve_coords_return result = go_kiosk_resolve_coords(location);
    if (result.r1 != NULL) {
        zend_throw_exception(spl_ce_RuntimeException, result.r1, 0);
        free(result.r1);
        RETURN_THROWS();
    }
    if (result.r0) {
        RETURN_ARR((zend_array *) result.r0);
    }

    RETURN_EMPTY_ARRAY();
}

PHP_FUNCTION(kiosk_fetch_weather)
{
    double lat = 0.0;
    double lon = 0.0;
    ZEND_PARSE_PARAMETERS_START(2, 2)
        Z_PARAM_DOUBLE(lat)
        Z_PARAM_DOUBLE(lon)
    ZEND_PARSE_PARAMETERS_END();

    struct go_kiosk_fetch_weather_return result = go_kiosk_fetch_weather((double) lat, (double) lon);
    if (result.r1 != NULL) {
        zend_throw_exception(spl_ce_RuntimeException, result.r1, 0);
        free(result.r1);
        RETURN_THROWS();
    }
    if (result.r0) {
        RETURN_ARR((zend_array *) result.r0);
    }

    RETURN_EMPTY_ARRAY();
}

PHP_FUNCTION(kiosk_fetch_tolls)
{
    ZEND_PARSE_PARAMETERS_NONE();

    struct go_kiosk_fetch_tolls_return result = go_kiosk_fetch_tolls();
    if (result.r1 != NULL) {
        zend_throw_exception(spl_ce_RuntimeException, result.r1, 0);
        free(result.r1);
        RETURN_THROWS();
    }
    if (result.r0) {
        RETURN_ARR((zend_array *) result.r0);
    }

    RETURN_EMPTY_ARRAY();
}

PHP_FUNCTION(kiosk_mercure_subscriptions)
{
    ZEND_PARSE_PARAMETERS_NONE();

    struct go_kiosk_mercure_subscriptions_return result = go_kiosk_mercure_subscriptions();
    if (result.r1 != NULL) {
        zend_throw_exception(spl_ce_RuntimeException, result.r1, 0);
        free(result.r1);
        RETURN_THROWS();
    }
    if (result.r0) {
        RETURN_ARR((zend_array *) result.r0);
    }

    RETURN_EMPTY_ARRAY();
}

PHP_FUNCTION(kiosk_mercure_publish)
{
    zend_string *topic = NULL;
    zend_string *data = NULL;
    zend_string *type = NULL;
    zend_string *id = NULL;
    ZEND_PARSE_PARAMETERS_START(4, 4)
        Z_PARAM_STR(topic)
        Z_PARAM_STR(data)
        Z_PARAM_STR(type)
        Z_PARAM_STR(id)
    ZEND_PARSE_PARAMETERS_END();

    struct go_kiosk_mercure_publish_return result = go_kiosk_mercure_publish(topic, data, type, id);
    if (result.r1 != NULL) {
        zend_throw_exception(spl_ce_RuntimeException, result.r1, 0);
        free(result.r1);
        RETURN_THROWS();
    }
    if (result.r0) {
        RETURN_ARR((zend_array *) result.r0);
    }

    RETURN_EMPTY_ARRAY();
}
