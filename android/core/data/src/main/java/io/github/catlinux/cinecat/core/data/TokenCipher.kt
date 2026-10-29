package io.github.catlinux.cinecat.core.data

interface TokenCipher {
    fun encrypt(data: ByteArray): ByteArray
    fun decrypt(data: ByteArray): ByteArray
}
