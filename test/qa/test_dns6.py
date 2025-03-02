from lib import (
    daemon,
    dns,
    info,
    logging,
    login,
    network,
)
import lib
import pytest
import random
import sh
import timeout_decorator


def setup_module(module):
    daemon.start()
    login.login_as("default")


def teardown_module(module):
    sh.nordvpn.logout("--persist-token")
    daemon.stop()


def setup_function(function):
    logging.log()


def teardown_function(function):
    logging.log(data=info.collect())
    logging.log()


@pytest.mark.parametrize("threat_protection_lite", lib.THREAT_PROTECTION_LITE)
@pytest.mark.flaky(reruns=2, reruns_delay=90)
@timeout_decorator.timeout(20)
def test_dns_connect(threat_protection_lite):
    lib.set_technology_and_protocol("openvpn", "udp", "off")
    lib.set_threat_protection_lite(threat_protection_lite)
    lib.set_ipv6("on")

    assert dns.is_unset()

    output = sh.nordvpn.connect(random.choice(lib.IPV6_SERVERS))

    print(output)
    assert lib.is_connect_successful(output)

    with lib.ErrorDefer(sh.nordvpn.disconnect):
        assert network.is_connected()
        assert dns.is_set_for(threat_protection_lite, "on")  # fails when connected over IPv4

    output = sh.nordvpn.disconnect()
    print(output)
    assert lib.is_disconnect_successful(output)
    assert network.is_disconnected()
    assert dns.is_unset()


@pytest.mark.flaky(reruns=2, reruns_delay=90)
@timeout_decorator.timeout(20)
def test_set_dns_connected():
    lib.set_technology_and_protocol("openvpn", "udp", "off")
    lib.set_threat_protection_lite("off")
    lib.set_ipv6("on")

    assert dns.is_unset()

    output = sh.nordvpn.connect(random.choice(lib.IPV6_SERVERS))
    print(output)
    assert lib.is_connect_successful(output)

    with lib.ErrorDefer(sh.nordvpn.disconnect):
        assert network.is_connected()
        assert dns.is_set_for("off", "on")

    lib.set_threat_protection_lite("on")

    with lib.ErrorDefer(sh.nordvpn.disconnect):
        assert network.is_connected()
        assert dns.is_set_for("on", "on")

    output = sh.nordvpn.disconnect()
    print(output)
    assert lib.is_disconnect_successful(output)
    assert dns.is_unset()
    assert network.is_disconnected()
