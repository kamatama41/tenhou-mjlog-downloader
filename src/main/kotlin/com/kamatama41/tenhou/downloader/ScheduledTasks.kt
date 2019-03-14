package com.kamatama41.tenhou.downloader

import org.slf4j.LoggerFactory
import org.springframework.scheduling.annotation.Scheduled
import org.springframework.stereotype.Component

@Component
class ScheduledTasks {
    val logger = LoggerFactory.getLogger(this::class.java)!!

//    @EnableScheduling class Config

    @Scheduled(cron = "*/5 * * * * *") // Once per 5 seconds
    fun test() {
        logger.info("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
    }
}
