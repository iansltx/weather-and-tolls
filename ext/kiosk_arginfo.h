/* This is a generated file, edit the .stub.php file instead.
 * Stub hash: 45ad409d07dfe72069d3360a97b5b34a7d102bba */

ZEND_BEGIN_ARG_WITH_RETURN_TYPE_INFO_EX(arginfo_kiosk_resolve_coords, 0, 1, IS_ARRAY, 0)
	ZEND_ARG_TYPE_INFO(0, location, IS_STRING, 0)
ZEND_END_ARG_INFO()

ZEND_BEGIN_ARG_WITH_RETURN_TYPE_INFO_EX(arginfo_kiosk_fetch_weather, 0, 2, IS_ARRAY, 0)
	ZEND_ARG_TYPE_INFO(0, lat, IS_DOUBLE, 0)
	ZEND_ARG_TYPE_INFO(0, lon, IS_DOUBLE, 0)
ZEND_END_ARG_INFO()

ZEND_BEGIN_ARG_WITH_RETURN_TYPE_INFO_EX(arginfo_kiosk_fetch_tolls, 0, 0, IS_ARRAY, 0)
ZEND_END_ARG_INFO()

#define arginfo_kiosk_mercure_subscriptions arginfo_kiosk_fetch_tolls

ZEND_BEGIN_ARG_WITH_RETURN_TYPE_INFO_EX(arginfo_kiosk_mercure_publish, 0, 4, IS_ARRAY, 0)
	ZEND_ARG_TYPE_INFO(0, topic, IS_STRING, 0)
	ZEND_ARG_TYPE_INFO(0, data, IS_STRING, 0)
	ZEND_ARG_TYPE_INFO(0, type, IS_STRING, 0)
	ZEND_ARG_TYPE_INFO(0, id, IS_STRING, 0)
ZEND_END_ARG_INFO()

ZEND_FUNCTION(kiosk_resolve_coords);
ZEND_FUNCTION(kiosk_fetch_weather);
ZEND_FUNCTION(kiosk_fetch_tolls);
ZEND_FUNCTION(kiosk_mercure_subscriptions);
ZEND_FUNCTION(kiosk_mercure_publish);

static const zend_function_entry ext_functions[] = {
	ZEND_FE(kiosk_resolve_coords, arginfo_kiosk_resolve_coords)
	ZEND_FE(kiosk_fetch_weather, arginfo_kiosk_fetch_weather)
	ZEND_FE(kiosk_fetch_tolls, arginfo_kiosk_fetch_tolls)
	ZEND_FE(kiosk_mercure_subscriptions, arginfo_kiosk_mercure_subscriptions)
	ZEND_FE(kiosk_mercure_publish, arginfo_kiosk_mercure_publish)
	ZEND_FE_END
};
