#!/usr/bin/env php
<?php
$message = file_get_contents( 'php://stdin' ) ;
// Decode to get original value
$test = json_decode($message, true);

if (!empty($test['error'])) {
    file_put_contents('php://stderr', $test['error']);
}

if (!empty($test['info'])) {
    file_put_contents('php://stdout', $test['info']);
}

if (!empty($test['exception'])) {
    throw new \Exception($test['exception']);
}

usleep($test['delay']);
exit($test['exitcode']);