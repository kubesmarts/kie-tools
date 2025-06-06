/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

import React, { useCallback, useEffect, useMemo, useState } from "react";
import { Form, FormGroup } from "@patternfly/react-core/dist/js/components/Form";
import { Text, TextContent, TextVariants } from "@patternfly/react-core/dist/js/components/Text";
import { TextInput } from "@patternfly/react-core/dist/js/components/TextInput";
import ExternalLinkAltIcon from "@patternfly/react-icons/dist/js/icons/external-link-alt-icon";
import InfoAltIcon from "@patternfly/react-icons/dist/js/icons/info-alt-icon";
import { Alert, AlertVariant } from "@patternfly/react-core/dist/js/components/Alert";
import { Button, ButtonVariant } from "@patternfly/react-core/dist/js/components/Button";
import { Spinner } from "@patternfly/react-core/dist/js/components/Spinner";
import { ValidatedOptions } from "@patternfly/react-core/dist/js/helpers";
import ExclamationCircleIcon from "@patternfly/react-icons/dist/js/icons/exclamation-circle-icon";
import { v4 as uuid } from "uuid";
import { getOctokitClient } from "../../github/Hooks";
import { useOnlineI18n } from "../../i18n";
import { useCancelableEffect } from "@kie-tools-core/react-hooks/dist/useCancelableEffect";
import { AccountsDispatchActionKind, AccountsSection, useAccounts, useAccountsDispatch } from "../AccountsContext";
import { useAuthSessions, useAuthSessionsDispatch } from "../../authSessions/AuthSessionsContext";
import { AuthSessionDescriptionList } from "../../authSessions/AuthSessionsList";
import { AUTH_SESSION_VERSION_NUMBER, GitAuthSession, isGitAuthSession } from "../../authSessions/AuthSessionApi";
import { PromiseStateStatus, usePromiseState } from "@kie-tools-core/react-hooks/dist/PromiseState";
import {
  GitAuthProvider,
  isSupportedGitAuthProviderType,
  SupportedGitAuthProviders,
} from "../../authProviders/AuthProvidersApi";
import { switchExpression } from "@kie-tools-core/switch-expression-ts";
import { AuthOptionsType, getBitbucketClient } from "../../bitbucket/Hooks";
import { useEnv } from "../../env/hooks/EnvContext";
import { getGitlabClient } from "../../gitlab/useGitlabClient";
import { HelperText, HelperTextItem } from "@patternfly/react-core/dist/js/components/HelperText";

export const GITHUB_OAUTH_TOKEN_SIZE = 40;
export const BITBUCKET_OAUTH_TOKEN_SIZE = 40;
export const GITLAB_OAUTH_TOKEN_SIZE = 40;

export const GITHUB_TOKENS_HOW_TO_URL =
  "https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token";
export const BITBUCKET_TOKENS_HOW_TO_URL = "https://support.atlassian.com/bitbucket-cloud/docs/create-an-app-password/";
export const GITLAB_TOKENS_HOW_TO_URL =
  "https://docs.gitlab.com/user/profile/personal_access_tokens/#create-a-personal-access-token";

export const GITHUB_OAUTH_SCOPES = ["repo", "gist"];
export const BITBUCKET_OAUTH_SCOPES = [
  "account",
  "repository",
  "repository:write",
  "repository:admin",
  "snippet",
  "snippet:write",
];

export const GITLAB_OAUTH_SCOPES = ["api", "read_user", "read_repository", "write_repository"];

export type AuthenticatedUserResponse = {
  headers: {
    scopes: string[];
  };
  data: Pick<GitAuthSession, "login" | "name" | "uuid" | "email">;
};

export function ConnectToGitSection(props: { authProvider: GitAuthProvider }) {
  const { i18n } = useOnlineI18n();
  const accounts = useAccounts();
  const { env } = useEnv();
  const accountsDispatch = useAccountsDispatch();
  const { authSessions } = useAuthSessions();
  const authSessionsDispatch = useAuthSessionsDispatch();

  const [usernameInput, setUsernameInput] = useState("");
  const [tokenInput, setTokenInput] = useState("");
  const [success, setSuccess] = useState(false);
  const [newAuthSession, setNewAuthSession] = usePromiseState<GitAuthSession>();
  const isGitSection =
    accounts.section === AccountsSection.CONNECT_TO_GITHUB ||
    accounts.section === AccountsSection.CONNECT_TO_BITBUCKET ||
    accounts.section === AccountsSection.CONNECT_TO_GITLAB;
  const selectedGitSession = isGitSection ? accounts.selectedAuthSession : undefined;

  useEffect(() => {
    if (selectedGitSession && isGitAuthSession(selectedGitSession)) {
      setUsernameInput(selectedGitSession.login);
    }
  }, [selectedGitSession]);

  useCancelableEffect(
    useCallback(
      ({ canceled }) => {
        const authProviderType = props.authProvider.type;
        if (!isSupportedGitAuthProviderType(authProviderType)) {
          return;
        }
        if (!tokenInput) {
          return;
        }

        if (success) {
          return;
        }

        setSuccess(false);
        setNewAuthSession({ loading: true });
        if (
          [...authSessions.values()]
            .filter(({ type }) => type === "git")
            .some(({ token }: GitAuthSession) => token === tokenInput)
        ) {
          setNewAuthSession({ error: i18n.connectToGitModal.auth.error.alreadyLoggedIn });
          return;
        }
        delay(600)
          .then(
            switchExpression<SupportedGitAuthProviders, () => Promise<AuthenticatedUserResponse>>(authProviderType, {
              bitbucket: () =>
                fetchAuthenticatedBitbucketUser(
                  env.KIE_SANDBOX_APP_NAME,
                  usernameInput,
                  tokenInput,
                  props.authProvider.domain,
                  env.KIE_SANDBOX_CORS_PROXY_URL,
                  props.authProvider.insecurelyDisableTlsCertificateValidation,
                  props.authProvider.disableEncoding
                ),
              github: () =>
                fetchAuthenticatedGitHubUser(
                  tokenInput,
                  props.authProvider.domain,
                  env.KIE_SANDBOX_CORS_PROXY_URL,
                  props.authProvider.insecurelyDisableTlsCertificateValidation,
                  props.authProvider.disableEncoding
                ),
              gitlab: () =>
                fetchAuthenticatedGitlabUser(
                  env.KIE_SANDBOX_APP_NAME,
                  tokenInput,
                  props.authProvider.domain,
                  env.KIE_SANDBOX_CORS_PROXY_URL,
                  props.authProvider.insecurelyDisableTlsCertificateValidation,
                  props.authProvider.disableEncoding
                ),
            })
          )
          .then((response) => {
            if (canceled.get()) {
              return;
            }
            const requiredScopes = switchExpression(authProviderType, {
              bitbucket: BITBUCKET_OAUTH_SCOPES,
              github: GITHUB_OAUTH_SCOPES,
              gitlab: GITLAB_OAUTH_SCOPES,
            });
            if (!response.headers.scopes.some((it) => requiredScopes.includes(it))) {
              setNewAuthSession({
                error: i18n.connectToGitModal.auth.error.oauthScopes(requiredScopes.toString()),
              });
            }

            const newAuthSession: GitAuthSession = {
              id: selectedGitSession?.id ?? uuid(),
              version: AUTH_SESSION_VERSION_NUMBER,
              token: tokenInput,
              type: "git",
              login: response.data.login,
              name: response.data.name ?? undefined,
              email: response.data.email ?? undefined,
              uuid: response.data.uuid ?? undefined,
              authProviderId: props.authProvider.id,
              createdAtDateISO: new Date().toISOString(),
            };

            setNewAuthSession({ data: newAuthSession });
            setSuccess(true);

            // batch updates
            if (selectedGitSession) {
              authSessionsDispatch.update(newAuthSession);
            } else {
              authSessionsDispatch.add(newAuthSession);
            }
          })
          .catch((e) => {
            if (canceled.get()) {
              return;
            }

            setNewAuthSession({ error: `${e}` });
          });
      },
      [
        props.authProvider.type,
        props.authProvider.domain,
        props.authProvider.insecurelyDisableTlsCertificateValidation,
        props.authProvider.disableEncoding,
        props.authProvider.id,
        tokenInput,
        success,
        setNewAuthSession,
        authSessions,
        i18n.connectToGitModal.auth.error,
        env.KIE_SANDBOX_APP_NAME,
        env.KIE_SANDBOX_CORS_PROXY_URL,
        usernameInput,
        authSessionsDispatch,
        selectedGitSession,
      ]
    )
  );

  const validation = useMemo(() => {
    if (!isSupportedGitAuthProviderType(props.authProvider.type)) {
      return {
        validated: ValidatedOptions.error,
        helperTextInvalidIcon: <ExclamationCircleIcon />,
        helperTextInvalid: "Unsupported Git Auth Provider. This should not happen.",
      };
    }
    if (!tokenInput) {
      return {
        validated: ValidatedOptions.default,
        helperTextIcon: <Spinner diameter={"1em"} style={{ display: "none" }} />,
        helperText: i18n.connectToGitModal[props.authProvider.type].validation.scopes.helper,
      };
    }

    const { status } = newAuthSession;
    switch (status) {
      case PromiseStateStatus.PENDING:
        return {
          validated: ValidatedOptions.default,
          helperTextIcon: <Spinner diameter={"1em"} />,
          helperText: i18n.connectToGitModal.status.loading,
        };
      case PromiseStateStatus.REJECTED:
        return {
          validated: ValidatedOptions.error,
          helperTextInvalid: newAuthSession.error.join(". "),
          helperTextInvalidIcon: <ExclamationCircleIcon />,
        };
      case PromiseStateStatus.RESOLVED:
        return { validated: ValidatedOptions.success };
      default:
        assertUnreachable(status);
    }
  }, [props.authProvider.type, tokenInput, newAuthSession, i18n.connectToGitModal]);

  const successPrimaryAction = useMemo(() => {
    if (
      (accounts.section !== AccountsSection.CONNECT_TO_GITHUB &&
        accounts.section !== AccountsSection.CONNECT_TO_BITBUCKET &&
        accounts.section !== AccountsSection.CONNECT_TO_GITLAB) ||
      !newAuthSession.data
    ) {
      return;
    }

    if (!accounts.onNewAuthSession) {
      return {
        action: () => accountsDispatch({ kind: AccountsDispatchActionKind.GO_HOME }),
        label: i18n.connectToGitModal.navigation.seeConnectedAccounts,
      };
    }

    return {
      action: () => accounts.onNewAuthSession?.(newAuthSession.data),
      label: i18n.connectToGitModal.navigation.continue,
    };
  }, [
    accounts,
    accountsDispatch,
    i18n.connectToGitModal.navigation.continue,
    i18n.connectToGitModal.navigation.seeConnectedAccounts,
    newAuthSession.data,
  ]);

  if (!props.authProvider?.type || !isSupportedGitAuthProviderType(props.authProvider?.type)) {
    return <></>;
  }
  return (
    <>
      {validation.validated === ValidatedOptions.success && (
        <>
          <Alert isPlain={true} isInline={true} variant={AlertVariant.success} title={`Successfully connected`}></Alert>
          <br />
          <br />
          <AuthSessionDescriptionList authSession={newAuthSession.data!} />
          <br />
          <br />
          <br />
          <Button variant={ButtonVariant.primary} onClick={successPrimaryAction?.action}>
            {successPrimaryAction?.label}
          </Button>
        </>
      )}
      {validation.validated !== ValidatedOptions.success && (
        <>
          <Form>
            {switchExpression(props.authProvider.type, {
              bitbucket: (
                <FormGroup
                  isRequired={true}
                  label={i18n.connectToGitModal[props.authProvider.type].form.username?.label}
                  fieldId={"username"}
                >
                  <TextInput
                    value={usernameInput}
                    autoComplete={"off"}
                    id="username-input"
                    name="usernameInput"
                    aria-describedby="username-text-input-helper"
                    placeholder={i18n.connectToGitModal[props.authProvider.type].form.username?.placeHolder}
                    onChange={(_event, v) => setUsernameInput(v.trim())}
                  />
                </FormGroup>
              ),
              default: <></>,
            })}
            <FormGroup
              isRequired={true}
              label={i18n.connectToGitModal[props.authProvider.type].form.token.label}
              fieldId={"github-pat"}
            >
              <TextInput
                value={obfuscate(tokenInput)}
                autoComplete={"off"}
                id="token-input"
                name="tokenInput"
                aria-describedby="token-text-input-helper"
                placeholder={i18n.connectToGitModal[props.authProvider.type].form.token.placeHolder}
                maxLength={tokenSize()}
                validated={validation.validated}
                onPaste={(e) => setTokenInput(e.clipboardData.getData("text/plain").slice(0, tokenSize()))}
                autoFocus={true}
              />
              <HelperText>
                {validation.validated === "error" ? (
                  <HelperTextItem variant="error" icon={validation.helperTextInvalidIcon}>
                    {validation.helperTextInvalid}
                  </HelperTextItem>
                ) : (
                  <HelperTextItem icon={validation.helperTextIcon}>{validation.helperText}</HelperTextItem>
                )}
              </HelperText>
            </FormGroup>
          </Form>
          <br />
          <h3>
            <a
              href={switchExpression(props.authProvider.type, {
                bitbucket: generateNewBitbucketTokenUrl(props.authProvider.domain),
                github: generateNewGitHubTokenUrl(props.authProvider.domain),
                gitlab: generateNewGitlabTokenUrl(props.authProvider.domain),
              })}
              target={"_blank"}
              rel={"noopener"}
            >
              {i18n.connectToGitModal[props.authProvider.type].footer.createNewToken}
              &nbsp;
              <ExternalLinkAltIcon className="pf-v5-u-mx-sm" />
            </a>
          </h3>
          <br />
          <br />
          <TextContent>
            <Text component={TextVariants.blockquote}>
              <InfoAltIcon />
              &nbsp;
              <span className="pf-v5-u-mr-sm">{i18n.connectToGitModal.auth.disclaimer}&nbsp;</span>
              <a
                href={switchExpression(props.authProvider.type, {
                  bitbucket: BITBUCKET_TOKENS_HOW_TO_URL,
                  github: GITHUB_TOKENS_HOW_TO_URL,
                  gitlab: GITLAB_TOKENS_HOW_TO_URL,
                })}
                target={"_blank"}
                rel={"noopener"}
              >
                {i18n.connectToGitModal[props.authProvider.type].body.learnMore}
                &nbsp;
                <ExternalLinkAltIcon className="pf-v5-u-mx-sm" />
              </a>
            </Text>
          </TextContent>
        </>
      )}
    </>
  );

  function tokenSize(): number | undefined {
    return switchExpression(props.authProvider.type, {
      bitbucket: BITBUCKET_OAUTH_TOKEN_SIZE,
      github: GITHUB_OAUTH_TOKEN_SIZE,
      gitlab: GITLAB_OAUTH_TOKEN_SIZE,
      default: -1,
    });
  }
}

export function obfuscate(token: string) {
  if (token.length <= 8) {
    return token;
  }

  const stars = new Array(token.length - 8).join("*");
  const pieceToObfuscate = token.substring(4, token.length - 4);
  return token.replace(pieceToObfuscate, stars);
}

export const generateNewBitbucketTokenUrl = (domain: string) => {
  return `https://${domain}/account/settings/app-passwords/`;
};
export const generateNewGitHubTokenUrl = (domain: string) => {
  return `https://${domain}/settings/tokens`;
};
export const generateNewGitlabTokenUrl = (domain: string): string =>
  `https://${domain}/-/user_settings/personal_access_tokens`;

function delay(ms: number) {
  return new Promise((res) => setTimeout(res, ms));
}

export function assertUnreachable(_x: never): never {
  throw new Error("Didn't expect to get here");
}

export const fetchAuthenticatedGitHubUser = async (
  githubToken: string,
  domain?: string,
  proxyUrl?: string,
  insecurelyDisableTlsCertificateValidation?: boolean,
  disableEncoding?: boolean
) => {
  const octokit = getOctokitClient({
    githubToken,
    domain,
    proxyUrl,
    insecurelyDisableTlsCertificateValidation,
    disableEncoding,
  });
  const response = await octokit.users.getAuthenticated();
  return {
    data: {
      name: response.data.name ?? undefined,
      login: response.data.login,
      email: response.data.email ?? undefined,
    },
    headers: {
      scopes: response.headers["x-oauth-scopes"]?.split(", ") ?? [],
    },
  };
};
export const fetchAuthenticatedBitbucketUser = async (
  appName: string,
  bitbucketUsername: string,
  bitbucketToken: string,
  domain?: string,
  proxyUrl?: string,
  insecurelyDisableTlsCertificateValidation?: boolean,
  disableEncoding?: boolean
) => {
  const bitbucketClient = getBitbucketClient({
    appName,
    domain,
    auth: {
      type: AuthOptionsType.BASIC,
      username: bitbucketUsername,
      password: bitbucketToken,
    },
    proxyUrl,
    insecurelyDisableTlsCertificateValidation,
    disableEncoding,
  });

  const response = await bitbucketClient.getAuthedUser();
  if (!response.ok) {
    const message = await response.text();
    throw new Error(
      `Error while authenticating user ${bitbucketUsername}: ${response.status} ${response.statusText} ${message}`
    );
  }
  const json = await response.json();
  return {
    data: {
      name: json?.display_name,
      login: bitbucketUsername,
      uuid: json.uuid,
    },
    headers: { scopes: response.headers.get("x-oauth-scopes")?.split(", ") ?? [] },
  };
};

export const fetchAuthenticatedGitlabUser = async (
  appName: string,
  gitlabToken: string,
  domain?: string,
  proxyUrl?: string,
  insecurelyDisableTlsCertificateValidation?: boolean,
  disableEncoding?: boolean
) => {
  const gitlabClient = getGitlabClient({
    appName,
    domain,
    token: gitlabToken,
    proxyUrl,
    insecurelyDisableTlsCertificateValidation,
    disableEncoding,
  });

  const response = await gitlabClient.getAuthedUser();

  if (!response.ok) {
    const message = await response.text();
    throw new Error(`While authenticating : ${response.status} ${response.statusText} ${message}`);
  }
  const jsonResponse = await response.json();

  return {
    data: {
      name: jsonResponse?.name,
      login: jsonResponse?.username,
      email: jsonResponse?.email,
    },
    // GitLab does not expose PAT scopes in the response
    headers: { scopes: [] },
  };
};
