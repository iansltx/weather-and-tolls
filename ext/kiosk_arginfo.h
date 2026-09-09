/* This is a generated file, edit the .stub.php file instead.
 * Stub hash: debbec4c04e6913eee51dca3b836c7996c3a63cd */

ZEND_BEGIN_ARG_INFO_EX(arginfo_class_Kiosk_Bridge___construct, 0, 0, 0)
ZEND_END_ARG_INFO()

ZEND_BEGIN_ARG_WITH_RETURN_TYPE_INFO_EX(arginfo_class_Kiosk_Bridge_resolveCoords, 0, 1, IS_ARRAY, 0)
	ZEND_ARG_TYPE_INFO(0, location, IS_STRING, 0)
ZEND_END_ARG_INFO()

ZEND_BEGIN_ARG_WITH_RETURN_TYPE_INFO_EX(arginfo_class_Kiosk_Bridge_fetchWeather, 0, 2, IS_ARRAY, 0)
	ZEND_ARG_TYPE_INFO(0, lat, IS_DOUBLE, 0)
	ZEND_ARG_TYPE_INFO(0, lon, IS_DOUBLE, 0)
ZEND_END_ARG_INFO()

ZEND_BEGIN_ARG_WITH_RETURN_TYPE_INFO_EX(arginfo_class_Kiosk_Bridge_fetchTolls, 0, 0, IS_ARRAY, 0)
ZEND_END_ARG_INFO()

#define arginfo_class_Kiosk_Bridge_mercureSubscriptions arginfo_class_Kiosk_Bridge_fetchTolls

ZEND_BEGIN_ARG_WITH_RETURN_TYPE_INFO_EX(arginfo_class_Kiosk_Bridge_mercurePublish, 0, 4, IS_ARRAY, 0)
	ZEND_ARG_TYPE_INFO(0, topic, IS_STRING, 0)
	ZEND_ARG_TYPE_INFO(0, data, IS_STRING, 0)
	ZEND_ARG_TYPE_INFO(0, updateType, IS_STRING, 0)
	ZEND_ARG_TYPE_INFO(0, id, IS_STRING, 0)
ZEND_END_ARG_INFO()

ZEND_METHOD(Kiosk_Bridge, __construct);
ZEND_METHOD(Kiosk_Bridge, resolveCoords);
ZEND_METHOD(Kiosk_Bridge, fetchWeather);
ZEND_METHOD(Kiosk_Bridge, fetchTolls);
ZEND_METHOD(Kiosk_Bridge, mercureSubscriptions);
ZEND_METHOD(Kiosk_Bridge, mercurePublish);

static const zend_function_entry class_Kiosk_Bridge_methods[] = {
	ZEND_ME(Kiosk_Bridge, __construct, arginfo_class_Kiosk_Bridge___construct, ZEND_ACC_PUBLIC)
	ZEND_ME(Kiosk_Bridge, resolveCoords, arginfo_class_Kiosk_Bridge_resolveCoords, ZEND_ACC_PUBLIC)
	ZEND_ME(Kiosk_Bridge, fetchWeather, arginfo_class_Kiosk_Bridge_fetchWeather, ZEND_ACC_PUBLIC)
	ZEND_ME(Kiosk_Bridge, fetchTolls, arginfo_class_Kiosk_Bridge_fetchTolls, ZEND_ACC_PUBLIC)
	ZEND_ME(Kiosk_Bridge, mercureSubscriptions, arginfo_class_Kiosk_Bridge_mercureSubscriptions, ZEND_ACC_PUBLIC)
	ZEND_ME(Kiosk_Bridge, mercurePublish, arginfo_class_Kiosk_Bridge_mercurePublish, ZEND_ACC_PUBLIC)
	ZEND_FE_END
};

static zend_class_entry *register_class_Kiosk_Bridge(void)
{
	zend_class_entry ce, *class_entry;

	INIT_NS_CLASS_ENTRY(ce, "Kiosk", "Bridge", class_Kiosk_Bridge_methods);
	class_entry = zend_register_internal_class(&ce);

	return class_entry;
}
