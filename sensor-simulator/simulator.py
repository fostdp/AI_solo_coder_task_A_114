#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
古代木拱桥结构监测 - 4G DTU 传感器模拟器 v2
模拟10座古代木拱桥的位移、应变、温湿度、振动、倾角传感器数据上报
支持HTTP REST API和MQTT双协议上报
支持不同车辆荷载脉冲、温湿度日/季节周期、随机断网
"""

import json
import os
import sys
import time
import random
import math
import traceback
from datetime import datetime, timedelta
from typing import List, Dict, Any, Optional, Callable

try:
    import requests
except ImportError:
    print("[WARN] requests not installed, HTTP mode disabled")
    requests = None

try:
    import paho.mqtt.client as mqtt_lib
except ImportError:
    mqtt_lib = None


VEHICLE_LOADS = {
    "pedestrian": 5.0,
    "ox_cart": 30.0,
    "horse_cart": 50.0,
    "military_carts": 80.0,
    "imperial_ceremony": 120.0,
}

LOAD_PATTERNS = [
    ("weekday_commute", [
        (7, 9, 0.9, "horse_cart"),
        (11, 13, 0.7, "ox_cart"),
        (17, 19, 0.9, "horse_cart"),
        (21, 23, 0.3, "pedestrian"),
    ]),
    ("weekend_festival", [
        (9, 11, 0.8, "horse_cart"),
        (11, 17, 1.0, "imperial_ceremony"),
        (17, 21, 0.85, "ox_cart"),
    ]),
    ("market_day", [
        (6, 8, 0.6, "ox_cart"),
        (8, 14, 0.95, "ox_cart"),
        (14, 18, 0.8, "horse_cart"),
    ]),
    ("military_convoy", [
        (2, 5, 0.9, "military_carts"),
        (20, 22, 0.8, "military_carts"),
    ]),
]


class BridgeSensorSimulator:
    def __init__(
        self,
        api_base_url: str = "http://backend:8080/api/v1",
        mqtt_host: Optional[str] = None,
        mqtt_port: int = 1883,
        mqtt_topic: str = "bridges/sensors",
        mqtt_username: Optional[str] = None,
        mqtt_password: Optional[str] = None,
        transport: str = "http",
    ):
        self.api_base_url = api_base_url.rstrip("/")
        self.transport = transport
        self.mqtt_host = mqtt_host
        self.mqtt_port = mqtt_port
        self.mqtt_topic = mqtt_topic
        self.mqtt_client = None
        self.mqtt_username = mqtt_username
        self.mqtt_password = mqtt_password
        self.network_available = True
        self.dtu_devices = self._init_dtu_devices()
        self.load_factor_cache: Dict[str, float] = {}

        if self.transport in ("mqtt", "both") and mqtt_host:
            self._init_mqtt_client()

    def _init_dtu_devices(self) -> List[Dict[str, Any]]:
        bridges = [
            {"bridge_id": 1, "name": "汴水虹桥", "dtu_id": "DTU-BIANSHUI-001", "typology": "贯木拱"},
            {"bridge_id": 2, "name": "永安桥", "dtu_id": "DTU-YONGAN-002", "typology": "叠梁拱"},
            {"bridge_id": 3, "name": "龙津桥", "dtu_id": "DTU-LONGJIN-003", "typology": "木拱廊桥"},
            {"bridge_id": 4, "name": "广济桥", "dtu_id": "DTU-GUANGJI-004", "typology": "贯木拱"},
            {"bridge_id": 5, "name": "万安桥", "dtu_id": "DTU-WANAN-005", "typology": "木拱廊桥"},
            {"bridge_id": 6, "name": "飞虹桥", "dtu_id": "DTU-FEIHONG-006", "typology": "叠梁拱"},
            {"bridge_id": 7, "name": "千乘桥", "dtu_id": "DTU-QIANSHENG-007", "typology": "木拱廊桥"},
            {"bridge_id": 8, "name": "安澜桥", "dtu_id": "DTU-ANLAN-008", "typology": "索桥"},
            {"bridge_id": 9, "name": "枫桥", "dtu_id": "DTU-FENGQIAO-009", "typology": "梁桥"},
            {"bridge_id": 10, "name": "灞桥", "dtu_id": "DTU-BAQIAO-010", "typology": "多跨梁桥"},
        ]

        for bridge in bridges:
            bridge["sensors"] = self._generate_sensors(bridge["bridge_id"])
            bridge["load_pattern_idx"] = (bridge["bridge_id"] - 1) % len(LOAD_PATTERNS)

        return bridges

    def _generate_sensors(self, bridge_id: int) -> List[Dict[str, Any]]:
        sensors = []

        sensor_types = [
            {"type": "displacement", "measurement": "位移", "unit": "mm", "count": 6},
            {"type": "strain", "measurement": "应变", "unit": "μɛ", "count": 8},
            {"type": "temperature", "measurement": "温度", "unit": "°C", "count": 4},
            {"type": "humidity", "measurement": "湿度", "unit": "%RH", "count": 3},
            {"type": "vibration", "measurement": "振动", "unit": "mm/s", "count": 4},
            {"type": "tilt", "measurement": "倾角", "unit": "°", "count": 2},
            {"type": "crack_width", "measurement": "裂缝宽度", "unit": "μm", "count": 2},
            {"type": "settlement", "measurement": "沉降", "unit": "mm", "count": 2},
        ]

        sensor_idx = 1
        for st in sensor_types:
            for i in range(st["count"]):
                sensor_code = f"S{bridge_id:02d}-{st['type'].upper()}-{sensor_idx:03d}"
                sensors.append({
                    "sensor_code": sensor_code,
                    "sensor_type": st["type"],
                    "measurement": st["measurement"],
                    "unit": st["unit"],
                    "base_value": self._get_base_value(st["type"], bridge_id),
                    "amplitude": self._get_amplitude(st["type"]),
                    "noise": self._get_noise(st["type"]),
                    "position": (i + 1) / (st["count"] + 1),
                    "range_min": self._get_range_min(st["type"]),
                    "range_max": self._get_range_max(st["type"]),
                    "drift": random.uniform(-0.002, 0.002),
                })
                sensor_idx += 1

        return sensors

    def _get_base_value(self, sensor_type: str, bridge_id: int) -> float:
        base_values = {
            "displacement": 5.0 + bridge_id * 0.3,
            "strain": 150.0 + bridge_id * 10,
            "temperature": 22.0,
            "humidity": 60.0,
            "vibration": 0.5,
            "tilt": 0.1,
            "crack_width": 25.0,
            "settlement": 0.5,
        }
        return base_values.get(sensor_type, 0.0)

    def _get_amplitude(self, sensor_type: str) -> float:
        amplitudes = {
            "displacement": 3.5,
            "strain": 80.0,
            "temperature": 15.0,
            "humidity": 30.0,
            "vibration": 0.6,
            "tilt": 0.1,
            "crack_width": 15.0,
            "settlement": 0.3,
        }
        return amplitudes.get(sensor_type, 0.0)

    def _get_noise(self, sensor_type: str) -> float:
        noises = {
            "displacement": 0.15,
            "strain": 6.0,
            "temperature": 0.4,
            "humidity": 1.2,
            "vibration": 0.08,
            "tilt": 0.006,
            "crack_width": 1.0,
            "settlement": 0.02,
        }
        return noises.get(sensor_type, 0.0)

    def _get_range_min(self, sensor_type: str) -> float:
        ranges = {
            "displacement": -10,
            "strain": -500,
            "temperature": -20,
            "humidity": 0,
            "vibration": 0,
            "tilt": -1.0,
            "crack_width": 0,
            "settlement": -10,
        }
        return ranges.get(sensor_type, 0.0)

    def _get_range_max(self, sensor_type: str) -> float:
        ranges = {
            "displacement": 30,
            "strain": 1500,
            "temperature": 50,
            "humidity": 100,
            "vibration": 5.0,
            "tilt": 1.0,
            "crack_width": 200,
            "settlement": 10,
        }
        return ranges.get(sensor_type, 100.0)

    def _get_vehicle_load_factor(self, current_time: datetime, pattern_name: str) -> float:
        pattern_data = dict(LOAD_PATTERNS)
        pattern_list = pattern_data.get(pattern_name, LOAD_PATTERNS[0][1])

        hour = current_time.hour + current_time.minute / 60.0
        total_factor = 0.0

        for (start_h, end_h, intensity, vehicle_type) in pattern_list:
            if start_h <= hour <= end_h:
                phase = (hour - start_h) / max(end_h - start_h, 0.1)
                pulse = math.sin(math.pi * phase)
                load_kn = VEHICLE_LOADS.get(vehicle_type, 10.0) / 50.0
                total_factor += intensity * pulse * load_kn

        return min(2.5, 1.0 + total_factor)

    def _seasonal_factor(self, current_time: datetime) -> float:
        day_of_year = current_time.timetuple().tm_yday
        return 1.0 + 0.15 * math.sin(2 * math.pi * (day_of_year - 80) / 365.0)

    def _simulate_network_status(self, current_time: datetime, force: bool = False) -> bool:
        if not force and not self.network_available:
            if random.random() < 0.02:
                self.network_available = True
                print(f"  [NETWORK] Restored at {current_time.strftime('%H:%M:%S')}")
            return self.network_available

        dropout_probs = [
            (2, 4, 0.03),
            (20, 22, 0.015),
            (-1, -1, 0.003),
        ]

        hour = current_time.hour
        prob = 0.003
        for (sh, eh, p) in dropout_probs:
            if sh == -1 or sh <= hour <= eh:
                prob = p
                break

        if random.random() < prob:
            if self.network_available:
                self.network_available = False
                print(f"  [NETWORK] Dropped out at {current_time.strftime('%H:%M:%S')}")
        return self.network_available

    def _simulate_sensor_value(self, sensor: Dict[str, Any], current_time: datetime, load_factor: float) -> float:
        hour_fraction = current_time.hour / 24.0
        day_cycle = math.sin(2 * math.pi * (hour_fraction - 0.25))

        seasonal = self._seasonal_factor(current_time)
        position_effect = math.sin(math.pi * sensor["position"])

        base = sensor["base_value"]
        amplitude = sensor["amplitude"]
        noise = sensor["noise"] * random.gauss(0, 1)
        drift = sensor["drift"] * int((current_time - datetime(2020, 1, 1)).total_seconds() / 3600)

        value = base + amplitude * day_cycle * position_effect * seasonal + drift + noise

        if sensor["sensor_type"] in ("strain", "displacement", "vibration", "tilt", "crack_width"):
            load_pulse = math.sin(2 * math.pi * current_time.minute / 12.0) * 0.15
            value *= load_factor * (1.0 + load_pulse)

        if sensor["sensor_type"] == "temperature":
            value = 22.0 + 12.0 * day_cycle * seasonal + random.gauss(0, 0.3)

        if sensor["sensor_type"] == "humidity":
            value = 60.0 + 25.0 * (-day_cycle) * seasonal + random.gauss(0, 1.0)
            value = max(20, min(98, value))

        return round(value, 4)

    def generate_dtu_payload(self, dtu_device: Dict[str, Any], current_time: datetime) -> Dict[str, Any]:
        pattern_name = LOAD_PATTERNS[dtu_device["load_pattern_idx"]][0]
        load_factor = self._get_vehicle_load_factor(current_time, pattern_name)

        readings = []

        for sensor in dtu_device["sensors"]:
            value = self._simulate_sensor_value(sensor, current_time, load_factor)
            quality = self._calculate_quality(value, sensor)

            readings.append({
                "sensor_code": sensor["sensor_code"],
                "value": value,
                "quality_flag": quality,
                "unit": sensor["unit"],
            })

        payload = {
            "dtu_device_id": dtu_device["dtu_id"],
            "timestamp": current_time.isoformat(),
            "readings": readings,
            "bridge_id": dtu_device["bridge_id"],
            "raw_data": {
                "signal_strength": round(random.uniform(-90, -52), 1) if self.network_available else round(random.uniform(-110, -95), 1),
                "battery_voltage": round(random.uniform(11.0, 14.0), 2),
                "module_temp": round(random.uniform(18, 40), 1),
                "load_factor": round(load_factor, 3),
                "pattern": pattern_name,
            }
        }

        return payload

    def _calculate_quality(self, value: float, sensor: Dict[str, Any]) -> int:
        r_min, r_max = sensor["range_min"], sensor["range_max"]
        if r_min <= value <= r_max:
            ratio = abs(value - sensor["base_value"]) / max(sensor["amplitude"], 0.001)
            if ratio < 1.5:
                return 0
            elif ratio < 2.5:
                return 1
            else:
                return 2
        else:
            return 2

    def _init_mqtt_client(self):
        if mqtt_lib is None:
            print("[WARN] paho.mqtt not installed, MQTT mode disabled")
            return

        client_id = f"dtu-simulator-{os.getpid()}-{int(time.time())}"
        self.mqtt_client = mqtt_lib.Client(client_id, clean_session=False)

        if self.mqtt_username:
            self.mqtt_client.username_pw_set(self.mqtt_username, self.mqtt_password)

        def on_connect(client, userdata, flags, rc):
            status = "OK" if rc == 0 else f"FAIL({rc})"
            print(f"[MQTT] Connected to {self.mqtt_host}:{self.mqtt_port} [{status}]")

        def on_disconnect(client, userdata, rc):
            print(f"[MQTT] Disconnected rc={rc}, will retry...")

        self.mqtt_client.on_connect = on_connect
        self.mqtt_client.on_disconnect = on_disconnect
        self.mqtt_client.reconnect_delay_set(min_delay=1, max_delay=30)

        try:
            self.mqtt_client.connect_async(self.mqtt_host, self.mqtt_port, keepalive=60)
            self.mqtt_client.loop_start()
        except Exception as e:
            print(f"[MQTT] Failed to connect: {e}")
            self.mqtt_client = None

    def send_via_http(self, payload: Dict[str, Any]) -> bool:
        if requests is None:
            return False
        try:
            url = f"{self.api_base_url}/sensors/dtu-ingest"
            response = requests.post(url, json=payload, timeout=(3, 8))
            if response.status_code == 200:
                result = response.json()
                return result.get("status") == "success"
            else:
                print(f"  [HTTP {response.status_code}] {response.text[:200]}")
                return False
        except requests.exceptions.RequestException as e:
            print(f"  [HTTP ERROR] {str(e)[:80]}")
            return False

    def send_via_mqtt(self, payload: Dict[str, Any]) -> bool:
        if self.mqtt_client is None:
            return False

        try:
            bridge_id = payload.get("bridge_id", 0)
            dtu_id = payload.get("dtu_device_id", "unknown")
            topic = f"{self.mqtt_topic}/{bridge_id}/{dtu_id}"

            info = self.mqtt_client.publish(
                topic,
                json.dumps(payload, ensure_ascii=False),
                qos=1,
                retain=False,
            )
            info.wait_for_publish(timeout=3)
            return info.rc == 0
        except Exception as e:
            print(f"  [MQTT ERROR] {str(e)[:80]}")
            return False

    def send_environmental_data(self, bridge_id: int, current_time: datetime):
        seasonal = self._seasonal_factor(current_time)
        day_cycle = math.sin(2 * math.pi * ((current_time.hour + current_time.minute / 60) - 0.25) / 24.0)

        temperature = 22.0 + 12.0 * day_cycle * seasonal + random.uniform(-0.3, 0.3)
        humidity = 60.0 + 25.0 * (-day_cycle) * seasonal + random.uniform(-1.5, 1.5)
        humidity = max(20, min(98, humidity))

        is_raining = day_cycle < -0.3 and random.random() < 0.25
        rainfall = 0.0
        if is_raining:
            rainfall = random.uniform(0.2, 8.0) * seasonal

        env_data = {
            "bridge_id": bridge_id,
            "timestamp": current_time.isoformat(),
            "temperature": round(temperature, 2),
            "humidity": round(humidity, 2),
            "wind_speed": round(random.uniform(0, 6) * seasonal, 2),
            "wind_direction": round(random.uniform(0, 360), 1),
            "rainfall": round(rainfall, 2),
            "atmospheric_pressure": round(1013 + random.uniform(-5, 5), 1),
        }

        if self.transport in ("http", "both") and requests is not None and self.network_available:
            try:
                requests.post(f"{self.api_base_url}/sensors/environmental", json=env_data, timeout=3)
            except Exception:
                pass

        return env_data

    def run_realtime_simulation(self, interval_seconds: int = 60, enable_outage: bool = True):
        print("=" * 70)
        print("古代木拱桥结构监测 - 实时4G DTU传感器模拟器 v2")
        print("=" * 70)
        print(f"API地址:    {self.api_base_url}")
        print(f"MQTT主机:   {self.mqtt_host}:{self.mqtt_port}")
        print(f"MQTT主题:   {self.mqtt_topic}")
        print(f"上报方式:   {self.transport.upper()}")
        print(f"上报间隔:   {interval_seconds}秒")
        print(f"模拟桥梁:   {len(self.dtu_devices)} 座")
        print(f"传感器:     {sum(len(b['sensors']) for b in self.dtu_devices)} 个/轮")
        print(f"断网模拟:   {'开启' if enable_outage else '关闭'}")
        print("按 Ctrl+C 停止模拟")
        print("=" * 70)

        sent_ok = 0
        sent_fail = 0
        cycle = 0

        try:
            while True:
                cycle += 1
                current_time = datetime.now()

                if enable_outage:
                    self._simulate_network_status(current_time)

                ok = 0
                fail = 0

                print(f"\n[{current_time.strftime('%Y-%m-%d %H:%M:%S')}] 第{cycle:4d}轮 | 网络:{'✅' if self.network_available else '❌'}")

                for dtu in self.dtu_devices:
                    payload = self.generate_dtu_payload(dtu, current_time)

                    result_flags = []

                    if self.transport in ("http", "both") and self.network_available:
                        h = self.send_via_http(payload)
                        result_flags.append(("HTTP", "✓" if h else "✗"))
                        if h:
                            ok += 1
                        else:
                            fail += 1

                    if self.transport in ("mqtt", "both"):
                        m = self.send_via_mqtt(payload)
                        result_flags.append(("MQTT", "✓" if m else "✗"))
                        if m:
                            ok += 1
                        else:
                            fail += 1

                    self.send_environmental_data(dtu["bridge_id"], current_time)

                    status_str = " ".join(f"{k}:{v}" for k, v in result_flags)
                    print(f"  {dtu['name'][:6]:<6} | {status_str} | {len(payload['readings'])}读数")

                sent_ok += ok
                sent_fail += fail
                total = sent_ok + sent_fail
                rate = (sent_ok / total * 100) if total > 0 else 0
                print(f"  累计: 成功{sent_ok} 失败{sent_fail} 成功率{rate:.1f}%")

                time.sleep(max(1, interval_seconds))

        except KeyboardInterrupt:
            print("\n\n" + "=" * 70)
            print(f"模拟已停止 | 发送成功 {sent_ok} / 失败 {sent_fail} | 共 {cycle} 轮")
            print("=" * 70)

    def run_batch_simulation(self, interval_seconds: int = 3600, duration_hours: int = 24):
        current_time = datetime.now().replace(minute=0, second=0, microsecond=0)
        total_steps = duration_hours * 3600 // interval_seconds

        print("=" * 70)
        print("古代木拱桥 - 批量模拟模式")
        print("=" * 70)
        print(f"开始时间: {current_time.strftime('%Y-%m-%d %H:%M:%S')}")
        print(f"步数:     {total_steps} ({duration_hours}h, 每{interval_seconds}s)")
        print("=" * 70)

        sent_ok = 0
        sent_fail = 0

        for i in range(total_steps):
            print(f"\n[第 {i+1}/{total_steps} 步] {current_time.strftime('%Y-%m-%d %H:%M:%S')}", end="")

            self.network_available = True

            for dtu in self.dtu_devices:
                payload = self.generate_dtu_payload(dtu, current_time)
                success = self.send_via_http(payload)
                if success:
                    sent_ok += 1
                else:
                    sent_fail += 1

                self.send_environmental_data(dtu["bridge_id"], current_time)

            current_time += timedelta(seconds=interval_seconds)
            rate = sent_ok / max(1, sent_ok + sent_fail) * 100
            print(f" | OK: {sent_ok} FAIL: {sent_fail}  ({rate:.1f}%)")

        print("\n" + "=" * 70)
        print(f"批量完成: 成功 {sent_ok} / 失败 {sent_fail}")
        print("=" * 70)

    def run_historical(self, days: int = 30):
        end_time = datetime.now().replace(minute=0, second=0, microsecond=0)
        start_time = end_time - timedelta(days=days)
        current_time = start_time
        total_hours = days * 24
        total_count = 0

        print("=" * 70)
        print(f"历史数据回填: 过去 {days} 天 ({total_hours} 小时)")
        print("=" * 70)

        last_day = None
        while current_time <= end_time:
            day_str = current_time.strftime("%Y-%m-%d")
            if day_str != last_day:
                print(f"  处理日期: {day_str}")
                last_day = day_str

            for dtu in self.dtu_devices:
                payload = self.generate_dtu_payload(dtu, current_time)
                self.send_via_http(payload)
                total_count += 1

            current_time += timedelta(hours=1)

        print(f"\n✅ 历史数据回填完成: {total_count} 条上报")
        return total_count


def main():
    import argparse

    parser = argparse.ArgumentParser(
        description="古代木拱桥4G DTU传感器模拟器 v2",
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument("--api-url",
                        default=os.environ.get("API_URL", "http://localhost:8080/api/v1"),
                        help="后端API地址")
    parser.add_argument("--transport",
                        choices=["http", "mqtt", "both"],
                        default=os.environ.get("TRANSPORT", "http"),
                        help="上报协议")
    parser.add_argument("--mqtt-host",
                        default=os.environ.get("MQTT_HOST", "mqtt-broker"),
                        help="MQTT Broker 主机")
    parser.add_argument("--mqtt-port", type=int,
                        default=int(os.environ.get("MQTT_PORT", "1883")),
                        help="MQTT端口")
    parser.add_argument("--mqtt-topic",
                        default=os.environ.get("MQTT_TOPIC", "bridges/sensors"),
                        help="MQTT主题前缀")
    parser.add_argument("--mqtt-username",
                        default=os.environ.get("MQTT_USER"),
                        help="MQTT用户名")
    parser.add_argument("--mqtt-password",
                        default=os.environ.get("MQTT_PASS"),
                        help="MQTT密码")
    parser.add_argument("--mode",
                        choices=["realtime", "batch", "historical"],
                        default=os.environ.get("MODE", "realtime"),
                        help="运行模式")
    parser.add_argument("--interval", type=int,
                        default=int(os.environ.get("INTERVAL", "60")),
                        help="上报间隔(秒),实时/批量用")
    parser.add_argument("--duration", type=int,
                        default=int(os.environ.get("DURATION", "24")),
                        help="批量模式时长(小时)")
    parser.add_argument("--days", type=int,
                        default=int(os.environ.get("DAYS", "30")),
                        help="历史模式天数")
    parser.add_argument("--no-outage", action="store_true",
                        help="关闭随机断网模拟")

    args = parser.parse_args()

    sim = BridgeSensorSimulator(
        api_base_url=args.api_url,
        mqtt_host=args.mqtt_host,
        mqtt_port=args.mqtt_port,
        mqtt_topic=args.mqtt_topic,
        mqtt_username=args.mqtt_username,
        mqtt_password=args.mqtt_password,
        transport=args.transport,
    )

    try:
        if args.mode == "realtime":
            sim.run_realtime_simulation(args.interval, not args.no_outage)
        elif args.mode == "batch":
            sim.run_batch_simulation(3600 // 1 if args.interval < 60 else args.interval, args.duration)
        elif args.mode == "historical":
            sim.run_historical(args.days)
    except Exception as e:
        print(f"\n[FATAL] {e}")
        traceback.print_exc()
        sys.exit(1)


if __name__ == "__main__":
    main()
