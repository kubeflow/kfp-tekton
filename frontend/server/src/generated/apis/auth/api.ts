/// <reference path="./custom.d.ts" />
// tslint:disable
/**
 * backend/api/v1/auth.proto
 * No description provided (generated by Swagger Codegen https://github.com/swagger-api/swagger-codegen)
 *
 * OpenAPI spec version: version not set
 *
 *
 * NOTE: This class is auto generated by the swagger code generator program.
 * https://github.com/swagger-api/swagger-codegen.git
 * Do not edit the class manually.
 */

import * as url from 'url';
import * as portableFetch from 'portable-fetch';
import { Configuration } from './configuration';

const BASE_PATH = 'http://localhost'.replace(/\/+$/, '');

/**
 *
 * @export
 */
export const COLLECTION_FORMATS = {
  csv: ',',
  ssv: ' ',
  tsv: '\t',
  pipes: '|',
};

/**
 *
 * @export
 * @interface FetchAPI
 */
export interface FetchAPI {
  (url: string, init?: any): Promise<Response>;
}

/**
 *
 * @export
 * @interface FetchArgs
 */
export interface FetchArgs {
  url: string;
  options: any;
}

/**
 *
 * @export
 * @class BaseAPI
 */
export class BaseAPI {
  protected configuration: Configuration;

  constructor(
    configuration?: Configuration,
    protected basePath: string = BASE_PATH,
    protected fetch: FetchAPI = portableFetch,
  ) {
    if (configuration) {
      this.configuration = configuration;
      this.basePath = configuration.basePath || this.basePath;
    }
  }
}

/**
 *
 * @export
 * @class RequiredError
 * @extends {Error}
 */
export class RequiredError extends Error {
  name: 'RequiredError';
  constructor(public field: string, msg?: string) {
    super(msg);
  }
}

/**
 * Type of resources in pipelines system.
 * @export
 * @enum {string}
 */
export enum AuthorizeRequestResources {
  UNASSIGNEDRESOURCES = <any>'UNASSIGNED_RESOURCES',
  VIEWERS = <any>'VIEWERS',
}

/**
 * Type of verbs that act on the resources.
 * @export
 * @enum {string}
 */
export enum AuthorizeRequestVerb {
  UNASSIGNEDVERB = <any>'UNASSIGNED_VERB',
  CREATE = <any>'CREATE',
  GET = <any>'GET',
  DELETE = <any>'DELETE',
}

/**
 * `Any` contains an arbitrary serialized protocol buffer message along with a URL that describes the type of the serialized message.  Protobuf library provides support to pack/unpack Any values in the form of utility functions or additional generated methods of the Any type.  Example 1: Pack and unpack a message in C++.      Foo foo = ...;     Any any;     any.PackFrom(foo);     ...     if (any.UnpackTo(&foo)) {       ...     }  Example 2: Pack and unpack a message in Java.      Foo foo = ...;     Any any = Any.pack(foo);     ...     if (any.is(Foo.class)) {       foo = any.unpack(Foo.class);     }   Example 3: Pack and unpack a message in Python.      foo = Foo(...)     any = Any()     any.Pack(foo)     ...     if any.Is(Foo.DESCRIPTOR):       any.Unpack(foo)       ...   Example 4: Pack and unpack a message in Go       foo := &pb.Foo{...}      any, err := anypb.New(foo)      if err != nil {        ...      }      ...      foo := &pb.Foo{}      if err := any.UnmarshalTo(foo); err != nil {        ...      }  The pack methods provided by protobuf library will by default use 'type.googleapis.com/full.type.name' as the type URL and the unpack methods only use the fully qualified type name after the last '/' in the type URL, for example \"foo.bar.com/x/y.z\" will yield type name \"y.z\".   JSON ==== The JSON representation of an `Any` value uses the regular representation of the deserialized, embedded message, with an additional field `@type` which contains the type URL. Example:      package google.profile;     message Person {       string first_name = 1;       string last_name = 2;     }      {       \"@type\": \"type.googleapis.com/google.profile.Person\",       \"firstName\": <string>,       \"lastName\": <string>     }  If the embedded message type is well-known and has a custom JSON representation, that representation will be embedded adding a field `value` which holds the custom JSON in addition to the `@type` field. Example (for message [google.protobuf.Duration][]):      {       \"@type\": \"type.googleapis.com/google.protobuf.Duration\",       \"value\": \"1.212s\"     }
 * @export
 * @interface ProtobufAny
 */
export interface ProtobufAny {
  /**
   * A URL/resource name that uniquely identifies the type of the serialized protocol buffer message. This string must contain at least one \"/\" character. The last segment of the URL's path must represent the fully qualified name of the type (as in `path/google.protobuf.Duration`). The name should be in a canonical form (e.g., leading \".\" is not accepted).  In practice, teams usually precompile into the binary all types that they expect it to use in the context of Any. However, for URLs which use the scheme `http`, `https`, or no scheme, one can optionally set up a type server that maps type URLs to message definitions as follows:  * If no scheme is provided, `https` is assumed. * An HTTP GET on the URL must yield a [google.protobuf.Type][]   value in binary format, or produce an error. * Applications are allowed to cache lookup results based on the   URL, or have them precompiled into a binary to avoid any   lookup. Therefore, binary compatibility needs to be preserved   on changes to types. (Use versioned type names to manage   breaking changes.)  Note: this functionality is not currently available in the official protobuf release, and it is not used for type URLs beginning with type.googleapis.com.  Schemes other than `http`, `https` (or the empty scheme) might be used with implementation specific semantics.
   * @type {string}
   * @memberof ProtobufAny
   */
  type_url?: string;
  /**
   * Must be a valid serialized protocol buffer of the above specified type.
   * @type {string}
   * @memberof ProtobufAny
   */
  value?: string;
}

/**
 *
 * @export
 * @interface V1Status
 */
export interface V1Status {
  /**
   *
   * @type {string}
   * @memberof V1Status
   */
  error?: string;
  /**
   *
   * @type {number}
   * @memberof V1Status
   */
  code?: number;
  /**
   *
   * @type {Array<ProtobufAny>}
   * @memberof V1Status
   */
  details?: Array<ProtobufAny>;
}

/**
 * AuthServiceApi - fetch parameter creator
 * @export
 */
export const AuthServiceApiFetchParamCreator = function(configuration?: Configuration) {
  return {
    /**
     *
     * @param {string} [namespace]
     * @param {'UNASSIGNED_RESOURCES' | 'VIEWERS'} [resources]
     * @param {'UNASSIGNED_VERB' | 'CREATE' | 'GET' | 'DELETE'} [verb]
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    authorize(
      namespace?: string,
      resources?: 'UNASSIGNED_RESOURCES' | 'VIEWERS',
      verb?: 'UNASSIGNED_VERB' | 'CREATE' | 'GET' | 'DELETE',
      options: any = {},
    ): FetchArgs {
      const localVarPath = `/apis/v1/auth`;
      const localVarUrlObj = url.parse(localVarPath, true);
      const localVarRequestOptions = Object.assign({ method: 'GET' }, options);
      const localVarHeaderParameter = {} as any;
      const localVarQueryParameter = {} as any;

      // authentication Bearer required
      if (configuration && configuration.apiKey) {
        const localVarApiKeyValue =
          typeof configuration.apiKey === 'function'
            ? configuration.apiKey('authorization')
            : configuration.apiKey;
        localVarHeaderParameter['authorization'] = localVarApiKeyValue;
      }

      if (namespace !== undefined) {
        localVarQueryParameter['namespace'] = namespace;
      }

      if (resources !== undefined) {
        localVarQueryParameter['resources'] = resources;
      }

      if (verb !== undefined) {
        localVarQueryParameter['verb'] = verb;
      }

      localVarUrlObj.query = Object.assign(
        {},
        localVarUrlObj.query,
        localVarQueryParameter,
        options.query,
      );
      // fix override query string Detail: https://stackoverflow.com/a/7517673/1077943
      delete localVarUrlObj.search;
      localVarRequestOptions.headers = Object.assign({}, localVarHeaderParameter, options.headers);

      return {
        url: url.format(localVarUrlObj),
        options: localVarRequestOptions,
      };
    },
  };
};

/**
 * AuthServiceApi - functional programming interface
 * @export
 */
export const AuthServiceApiFp = function(configuration?: Configuration) {
  return {
    /**
     *
     * @param {string} [namespace]
     * @param {'UNASSIGNED_RESOURCES' | 'VIEWERS'} [resources]
     * @param {'UNASSIGNED_VERB' | 'CREATE' | 'GET' | 'DELETE'} [verb]
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    authorize(
      namespace?: string,
      resources?: 'UNASSIGNED_RESOURCES' | 'VIEWERS',
      verb?: 'UNASSIGNED_VERB' | 'CREATE' | 'GET' | 'DELETE',
      options?: any,
    ): (fetch?: FetchAPI, basePath?: string) => Promise<any> {
      const localVarFetchArgs = AuthServiceApiFetchParamCreator(configuration).authorize(
        namespace,
        resources,
        verb,
        options,
      );
      return (fetch: FetchAPI = portableFetch, basePath: string = BASE_PATH) => {
        return fetch(basePath + localVarFetchArgs.url, localVarFetchArgs.options).then(response => {
          if (response.status >= 200 && response.status < 300) {
            return response.json();
          } else {
            throw response;
          }
        });
      };
    },
  };
};

/**
 * AuthServiceApi - factory interface
 * @export
 */
export const AuthServiceApiFactory = function(
  configuration?: Configuration,
  fetch?: FetchAPI,
  basePath?: string,
) {
  return {
    /**
     *
     * @param {string} [namespace]
     * @param {'UNASSIGNED_RESOURCES' | 'VIEWERS'} [resources]
     * @param {'UNASSIGNED_VERB' | 'CREATE' | 'GET' | 'DELETE'} [verb]
     * @param {*} [options] Override http request option.
     * @throws {RequiredError}
     */
    authorize(
      namespace?: string,
      resources?: 'UNASSIGNED_RESOURCES' | 'VIEWERS',
      verb?: 'UNASSIGNED_VERB' | 'CREATE' | 'GET' | 'DELETE',
      options?: any,
    ) {
      return AuthServiceApiFp(configuration).authorize(
        namespace,
        resources,
        verb,
        options,
      )(fetch, basePath);
    },
  };
};

/**
 * AuthServiceApi - object-oriented interface
 * @export
 * @class AuthServiceApi
 * @extends {BaseAPI}
 */
export class AuthServiceApi extends BaseAPI {
  /**
   *
   * @param {string} [namespace]
   * @param {'UNASSIGNED_RESOURCES' | 'VIEWERS'} [resources]
   * @param {'UNASSIGNED_VERB' | 'CREATE' | 'GET' | 'DELETE'} [verb]
   * @param {*} [options] Override http request option.
   * @throws {RequiredError}
   * @memberof AuthServiceApi
   */
  public authorize(
    namespace?: string,
    resources?: 'UNASSIGNED_RESOURCES' | 'VIEWERS',
    verb?: 'UNASSIGNED_VERB' | 'CREATE' | 'GET' | 'DELETE',
    options?: any,
  ) {
    return AuthServiceApiFp(this.configuration).authorize(
      namespace,
      resources,
      verb,
      options,
    )(this.fetch, this.basePath);
  }
}
