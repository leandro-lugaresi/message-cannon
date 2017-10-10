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

if (!empty($test['sleep'])) {
    usleep($test['sleep']);
}

exit($test['exitcode']);