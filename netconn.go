/*
 * Copyright (c) 2021 IBM Corp and others.
 *
 * All rights reserved. This program and the accompanying materials
 * are made available under the terms of the Eclipse Public License v2.0
 * and Eclipse Distribution License v1.0 which accompany this distribution.
 *
 * The Eclipse Public License is available at
 *    https://www.eclipse.org/legal/epl-2.0/
 * and the Eclipse Distribution License is available at
 *   http://www.eclipse.org/org/documents/edl-v10.php.
 *
 * Contributors:
 *    Seth Hoenig
 *    Allan Stockdill-Mander
 *    Mike Robertson
 *    MAtt Brittan
 */

package mqtt

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"golang.org/x/net/proxy"
)

//
// This just establishes the network connection; once established the type of connection should be irrelevant
//

// openConnection opens a network connection using the protocol indicated in the URL.
// Does not carry out any MQTT specific handshakes.
func openConnection(uri *url.URL, tlsc *tls.Config, timeout time.Duration, headers http.Header, websocketOptions *WebsocketOptions, dialer *net.Dialer) (net.Conn, error) {
	switch uri.Scheme {
	case "ws":
		conn, err := NewWebsocket(uri.String(), nil, timeout, headers, websocketOptions)
		return conn, err
	case "wss":
		conn, err := NewWebsocket(uri.String(), tlsc, timeout, headers, websocketOptions)
		return conn, err
	case "mqtt", "tcp":
		allProxy := os.Getenv("all_proxy")
		if len(allProxy) == 0 {
			fmt.Println("NO PROXY! dialing")
			conn, err := dialer.Dial("tcp", uri.Host)
			fmt.Println("NO PROXY! donedial")
			if err != nil {
				fmt.Println("NO PROXY! err", err)
				return nil, err
			}
			fmt.Println("NO PROXY! happy conn")
			return conn, nil
		}
		proxyDialer := proxy.FromEnvironment()

		fmt.Println("YES PROXY! dialing", proxyDialer)
		conn, err := proxyDialer.Dial("tcp", uri.Host)
		fmt.Println("YES PROXY! donedial")
		if err != nil {
			fmt.Println("YES PROXY! err", err)
			return nil, err
		}
		fmt.Println("YES PROXY! happy conn", conn)
		return conn, nil
	case "unix":
		var conn net.Conn
		var err error

		// this check is preserved for compatibility with older versions
		// which used uri.Host only (it works for local paths, e.g. unix://socket.sock in current dir)
		if len(uri.Host) > 0 {
			conn, err = dialer.Dial("unix", uri.Host)
		} else {
			conn, err = dialer.Dial("unix", uri.Path)
		}

		if err != nil {
			return nil, err
		}
		return conn, nil
	case "ssl", "tls", "mqtts", "mqtt+ssl", "tcps":
		allProxy := os.Getenv("all_proxy")
		if len(allProxy) == 0 {
			conn, err := tls.DialWithDialer(dialer, "tcp", uri.Host, tlsc)
			if err != nil {
				return nil, err
			}
			return conn, nil
		}
		proxyDialer := proxy.FromEnvironment()
		conn, err := proxyDialer.Dial("tcp", uri.Host)
		if err != nil {
			return nil, err
		}

		tlsConn := tls.Client(conn, tlsc)

		err = tlsConn.Handshake()
		if err != nil {
			_ = conn.Close()
			return nil, err
		}

		return tlsConn, nil
	}
	return nil, errors.New("unknown protocol")
}
