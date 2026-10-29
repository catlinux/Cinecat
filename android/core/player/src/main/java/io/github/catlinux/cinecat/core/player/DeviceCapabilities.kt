package io.github.catlinux.cinecat.core.player

import android.content.Context
import android.media.MediaCodecInfo
import android.media.MediaCodecList
import androidx.media3.common.C
import androidx.media3.common.util.UnstableApi
import androidx.media3.exoplayer.audio.AudioCapabilities

enum class VideoCodec(val token: String) {
    H264("h264"),
    H264_HI10("h264hi10"),
    HEVC("hevc"),
    HEVC10("hevc10"),
    AV1("av1"),
    VP9("vp9")
}

enum class AudioCodec(val token: String) {
    AAC("aac"),
    MP3("mp3"),
    AC3("ac3"),
    EAC3("eac3"),
    DTS("dts"),
    TRUEHD("truehd"),
    OPUS("opus"),
    FLAC("flac")
}

fun capsString(
    video: Set<VideoCodec>,
    audio: Set<AudioCodec>,
    multichannel: Boolean
): String {
    val tokens = buildList {
        add("mkv")
        add("mp4")
        VideoCodec.entries.forEach { codec ->
            if (codec in video) add(codec.token)
        }
        AudioCodec.entries.forEach { codec ->
            if (codec in audio) add(codec.token)
        }
        if (multichannel) add("mch")
    }
    return tokens.joinToString(",")
}

@UnstableApi
fun detectCapabilities(context: Context): String {
    val video = mutableSetOf<VideoCodec>()
    val audio = mutableSetOf<AudioCodec>()
    val codecList = MediaCodecList(MediaCodecList.REGULAR_CODECS)

    for (info in codecList.codecInfos) {
        if (info.isEncoder) {
            continue
        }
        for (type in info.supportedTypes) {
            when (type) {
                "video/avc" -> {
                    video += VideoCodec.H264
                    if (info.supportsProfile(type, MediaCodecInfo.CodecProfileLevel.AVCProfileHigh10)) {
                        video += VideoCodec.H264_HI10
                    }
                }
                "video/hevc" -> {
                    video += VideoCodec.HEVC
                    if (info.supportsProfile(type, MediaCodecInfo.CodecProfileLevel.HEVCProfileMain10)) {
                        video += VideoCodec.HEVC10
                    }
                }
                "video/av01" -> video += VideoCodec.AV1
                "video/x-vnd.on2.vp9" -> video += VideoCodec.VP9
                "audio/mp4a-latm" -> audio += AudioCodec.AAC
                "audio/mpeg" -> audio += AudioCodec.MP3
                "audio/ac3" -> audio += AudioCodec.AC3
                "audio/eac3" -> audio += AudioCodec.EAC3
                "audio/vnd.dts" -> audio += AudioCodec.DTS
                "audio/true-hd" -> audio += AudioCodec.TRUEHD
                "audio/opus" -> audio += AudioCodec.OPUS
                "audio/flac" -> audio += AudioCodec.FLAC
            }
        }
    }

    // ExoPlayer decodifica estos códecs por software aunque el equipo no los anuncie.
    audio += AudioCodec.AAC
    audio += AudioCodec.MP3
    audio += AudioCodec.OPUS
    audio += AudioCodec.FLAC

    // AC-3, E-AC-3, DTS y TrueHD solo son válidos si el receptor HDMI los
    // acepta sin decodificar; AudioCapabilities consulta esa cadena.
    val audioCapabilities = AudioCapabilities.getCapabilities(context)
    if (audioCapabilities.supportsEncoding(C.ENCODING_AC3)) {
        audio += AudioCodec.AC3
    }
    if (audioCapabilities.supportsEncoding(C.ENCODING_E_AC3)) {
        audio += AudioCodec.EAC3
    }
    if (audioCapabilities.supportsEncoding(C.ENCODING_DTS)) {
        audio += AudioCodec.DTS
    }
    if (audioCapabilities.supportsEncoding(C.ENCODING_DOLBY_TRUEHD)) {
        audio += AudioCodec.TRUEHD
    }

    return capsString(video, audio, multichannel = true)
}

private fun MediaCodecInfo.supportsProfile(mime: String, profile: Int): Boolean =
    runCatching {
        getCapabilitiesForType(mime).profileLevels.any { level -> level.profile == profile }
    }.getOrDefault(false)
