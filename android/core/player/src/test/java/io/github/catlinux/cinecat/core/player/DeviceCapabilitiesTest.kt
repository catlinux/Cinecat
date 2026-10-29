package io.github.catlinux.cinecat.core.player

import org.junit.Assert.assertEquals
import org.junit.Test

class DeviceCapabilitiesTest {
    @Test
    fun includesContainersAndOnlySelectedCodecs() {
        assertEquals(
            "mkv,mp4,h264,aac",
            capsString(
                video = setOf(VideoCodec.H264),
                audio = setOf(AudioCodec.AAC),
                multichannel = false
            )
        )
    }

    @Test
    fun realTelevisionKeepsAStableOrder() {
        assertEquals(
            "mkv,mp4,h264,hevc,vp9,aac,flac",
            capsString(
                video = setOf(VideoCodec.VP9, VideoCodec.HEVC, VideoCodec.H264),
                audio = setOf(AudioCodec.FLAC, AudioCodec.AAC),
                multichannel = false
            )
        )
    }

    @Test
    fun passthroughAndMultichannelAreDeclared() {
        assertEquals(
            "mkv,mp4,h264,hevc,aac,ac3,eac3,mch",
            capsString(
                video = setOf(VideoCodec.HEVC, VideoCodec.H264),
                audio = setOf(
                    AudioCodec.EAC3,
                    AudioCodec.AAC,
                    AudioCodec.AC3
                ),
                multichannel = true
            )
        )
    }

    @Test
    fun hevc10AndHigh10AreIndependentTokens() {
        assertEquals(
            "mkv,mp4,h264,h264hi10,hevc,hevc10,av1,vp9,aac,mp3,ac3,eac3,dts,truehd,opus,flac,mch",
            capsString(
                video = setOf(
                    VideoCodec.AV1,
                    VideoCodec.H264_HI10,
                    VideoCodec.HEVC10,
                    VideoCodec.H264,
                    VideoCodec.HEVC,
                    VideoCodec.VP9
                ),
                audio = setOf(
                    AudioCodec.FLAC,
                    AudioCodec.OPUS,
                    AudioCodec.AC3,
                    AudioCodec.EAC3,
                    AudioCodec.DTS,
                    AudioCodec.TRUEHD,
                    AudioCodec.MP3,
                    AudioCodec.AAC
                ),
                multichannel = true
            )
        )
    }

    @Test
    fun emptyCapabilitiesStillDeclareContainers() {
        assertEquals(
            "mkv,mp4,mch",
            capsString(emptySet(), emptySet(), multichannel = true)
        )
    }
}
