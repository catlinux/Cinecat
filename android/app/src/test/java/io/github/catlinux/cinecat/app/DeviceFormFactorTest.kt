package io.github.catlinux.cinecat.app

import android.content.res.Configuration
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class DeviceFormFactorTest {
    @Test
    fun televisionUsesTvUi() {
        assertTrue(isTelevision(Configuration.UI_MODE_TYPE_TELEVISION))
    }

    @Test
    fun televisionCapsDeclareMultichannel() {
        var multichannel: Boolean? = null
        val caps = calculateCaps(Configuration.UI_MODE_TYPE_TELEVISION) { requested ->
            multichannel = requested
            if (requested) "mkv,mp4,mch" else "mkv,mp4"
        }
        assertEquals(true, multichannel)
        assertTrue(caps.contains("mch"))
    }

    @Test
    fun mobileCapsDoNotDeclareMultichannel() {
        var multichannel: Boolean? = null
        val caps = calculateCaps(Configuration.UI_MODE_TYPE_NORMAL) { requested ->
            multichannel = requested
            if (requested) "mkv,mp4,mch" else "mkv,mp4"
        }
        assertEquals(false, multichannel)
        assertFalse(caps.contains("mch"))
    }

    @Test
    fun nonTelevisionFormFactorsUseMobileUi() {
        val values = intArrayOf(
            Configuration.UI_MODE_TYPE_NORMAL,
            Configuration.UI_MODE_TYPE_DESK,
            Configuration.UI_MODE_TYPE_CAR,
            Configuration.UI_MODE_TYPE_APPLIANCE,
            Configuration.UI_MODE_TYPE_WATCH,
            Configuration.UI_MODE_TYPE_VR_HEADSET
        )
        values.forEach { assertFalse(isTelevision(it)) }
    }
}
